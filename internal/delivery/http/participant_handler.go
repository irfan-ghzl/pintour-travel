package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v4"

	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

type ParticipantHandler struct{ svc *participantsvc.Service }

func NewParticipantHandler(svc *participantsvc.Service) *ParticipantHandler {
	return &ParticipantHandler{svc}
}

// ListParticipants godoc
// @Summary      Daftar peserta (admin)
// @Tags         participants
// @Security     BearerAuth
// @Param        batch_id query string false "Filter batch ID"
// @Param        search query string false "Cari nama/WA"
// @Param        page query int false "Halaman" default(1)
// @Success      200 {object} map[string]interface{}
// @Router       /admin/participants [get]
func (h *ParticipantHandler) ListParticipants(c echo.Context) error {
	f := domainParticipant.Filter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryPageSize(c, "per_page", 20),
	}
	if v := c.QueryParam("batch_id"); v != "" {
		f.BatchID = &v
	}
	if v := c.QueryParam("search"); v != "" {
		f.Search = &v
	}
	// A konsultan only sees participants converted from their own leads.
	if claimRole(c) == "konsultan" {
		uid := claimUserID(c)
		f.AssignedTo = &uid
	}
	list, total, err := h.svc.ListParticipants(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, pageResponse(list, total, f.Page, f.PerPage))
}

// GetParticipant godoc
// @Summary      Detail peserta
// @Tags         participants
// @Security     BearerAuth
// @Param        id path string true "Participant ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/participants/{id} [get]
func (h *ParticipantHandler) GetParticipant(c echo.Context) error {
	p, err := h.svc.GetParticipant(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "peserta tidak ditemukan")
	}
	return c.JSON(http.StatusOK, ok(p))
}

// TODO(ocr-v2.0-F3): SetNIK — aktifkan route-nya di router.go ketika billing on.
// SetNIK godoc
// @Summary      Set NIK peserta (mis. dari hasil OCR) (v2.0 F3)
// @Tags         participants
// @Security     BearerAuth
// @Param        id path string true "Participant ID"
// @Accept       json
// @Success      200 {object} map[string]interface{}
// @Router       /admin/participants/{id}/nik [patch]
func (h *ParticipantHandler) SetNIK(c echo.Context) error {
	var body struct {
		NIK string `json:"nik" validate:"required,max=20"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "nik harus diisi")
	}
	if err := h.svc.SetNIK(c.Request().Context(), c.Param("id"), body.NIK); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"message": "NIK peserta diperbarui"}))
}

// ConvertLead godoc
// @Summary      Konversi leads 'deal' menjadi peserta aktif (FR-CRM-06)
// @Tags         participants
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      201 {object} map[string]interface{}
// @Router       /admin/participants/convert [post]
func (h *ParticipantHandler) ConvertLead(c echo.Context) error {
	var body struct {
		LeadID   string `json:"lead_id" validate:"required"`
		BatchID  string `json:"batch_id" validate:"required"`
		RoomType string `json:"room_type" validate:"omitempty,oneof=single double triple"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "lead_id dan batch_id harus diisi")
	}
	if body.RoomType == "" {
		body.RoomType = "double"
	}

	res, err := h.svc.ConvertFromLead(c.Request().Context(), body.LeadID, body.BatchID, body.RoomType, claimUserID(c))
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, errResponse("CONVERT_FAILED", err.Error()))
	}
	// Returning customer (F1): reused the existing portal account, no new password.
	message := "Peserta berhasil dibuat. Kirimkan password sementara via WA."
	if res.ReusedAccount {
		message = "Peserta lama terdeteksi — akun portal yang sudah ada digunakan. " +
			"Password lama tetap berlaku, tidak perlu kirim password baru."
	}
	return c.JSON(http.StatusCreated, ok(map[string]interface{}{
		"participant":    res.Participant,
		"temp_password":  res.TempPassword,
		"reused_account": res.ReusedAccount,
		"message":        message,
	}))
}
