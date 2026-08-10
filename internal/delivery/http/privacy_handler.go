package httpdelivery

// The admin side of §25.5. A participant's erasure request is a clock the
// business is running against — 14 working days under UU PDP Pasal 46 — and a
// request nobody can see is a clock nobody is watching.

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/privacy"
)

// PrivacyHandler serves the erasure queue and the decisions taken on it.
type PrivacyHandler struct {
	deletions privacy.Repository
}

func NewPrivacyHandler(deletions privacy.Repository) *PrivacyHandler {
	return &PrivacyHandler{deletions: deletions}
}

// ListDeletionRequests godoc
//
// Requests waiting on someone come first, because those are the ones with time
// running against them. `days_waiting` is served alongside each row so a
// reviewer sees the deadline without doing arithmetic.
//
//	@Summary  Antrean permintaan penghapusan data peserta (§25.5)
//	@Tags     privacy
//	@Security BearerAuth
//	@Param    status query string false "Filter status (menunggu/selesai/ditolak)"
//	@Success  200 {object} map[string]interface{}
//	@Router   /admin/privacy/deletion-requests [get]
func (h *PrivacyHandler) ListDeletionRequests(c echo.Context) error {
	if h.deletions == nil {
		return serverErr(c, errDeletionsUnavailable)
	}
	list, err := h.deletions.List(c.Request().Context(), c.QueryParam("status"))
	if err != nil {
		return serverErr(c, err)
	}

	out := make([]map[string]any, 0, len(list))
	overdue := 0
	for i := range list {
		r := &list[i]
		waiting := r.DaysWaiting()
		if r.IsOpen() && waiting > 14 {
			overdue++
		}
		out = append(out, map[string]any{
			"request":      r,
			"days_waiting": waiting,
			"is_open":      r.IsOpen(),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    out,
		"meta":    map[string]int{"total": len(out), "melewati_batas": overdue},
	})
}

// ProcessDeletionRequest godoc
//
// Approving anonymises the participant: their name, phone, email, NIK, and
// portal credential are overwritten and their uploaded documents are removed.
// The participant row and its invoices stay — §25.4 anonymises participant data
// rather than deleting it, and an invoice is a financial record that §25.5's
// "kecuali yang wajib dipertahankan secara hukum" covers.
//
// Rejecting records the refusal and its reason, which is what makes a refusal
// answerable later. A participant with a departure still ahead is the case this
// exists for: the trip needs the identity documents the request would remove.
//
//	@Summary  Setujui atau tolak permintaan penghapusan data (§25.5)
//	@Tags     privacy
//	@Security BearerAuth
//	@Param    id path string true "Deletion request ID"
//	@Accept   json
//	@Success  200 {object} map[string]interface{}
//	@Router   /admin/privacy/deletion-requests/{id}/process [post]
func (h *PrivacyHandler) ProcessDeletionRequest(c echo.Context) error {
	if h.deletions == nil {
		return serverErr(c, errDeletionsUnavailable)
	}
	var body struct {
		Decision string `json:"decision" validate:"required,oneof=setujui tolak"`
		Notes    string `json:"notes" validate:"required_if=Decision tolak,max=1000"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "decision harus 'setujui' atau 'tolak'")
	}

	ctx := c.Request().Context()
	id := c.Param("id")

	var err error
	if body.Decision == "tolak" {
		err = h.deletions.Reject(ctx, id, claimUserID(c), body.Notes)
	} else {
		err = h.deletions.Anonymise(ctx, id, claimUserID(c), body.Notes)
	}
	switch {
	case errors.Is(err, privacy.ErrAlreadyProcessed):
		// Two reviewers working the same queue. The second is told the decision
		// was already taken rather than being allowed to take it again.
		return c.JSON(http.StatusConflict,
			errResponse("ALREADY_PROCESSED", "Permintaan ini sudah diproses sebelumnya"))
	case err != nil:
		return serverErr(c, err)
	}

	message := "Data peserta telah dianonimkan dan dokumennya dihapus."
	if body.Decision == "tolak" {
		message = "Permintaan ditolak, alasan tercatat."
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"message": message}))
}
