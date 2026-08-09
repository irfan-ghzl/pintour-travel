package httpdelivery

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/irfan-ghzl/pintour-travel/internal/config"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

type DocumentHandler struct {
	docs         document.Repository
	reqs         document.CountryRequirementRepository
	participants participant.Repository
	airport      airport.Repository
	fonnte       *service.FonnteService
	email        *service.EmailService
	// TODO(ocr-v2.0-F3): re-enable fields below when GCP Vision billing active
	ocr     *service.OCRService
	ocrRepo document.OCRResultRepository
}

func NewDocumentHandler(
	docs document.Repository,
	reqs document.CountryRequirementRepository,
	participants participant.Repository,
	airportRepo airport.Repository,
	fonnte *service.FonnteService,
	email *service.EmailService,
	ocr *service.OCRService,
	ocrRepo document.OCRResultRepository,
) *DocumentHandler {
	return &DocumentHandler{docs: docs, reqs: reqs, participants: participants, airport: airportRepo, fonnte: fonnte, email: email, ocr: ocr, ocrRepo: ocrRepo}
}

// triggerOCR runs OCR asynchronously for a freshly-uploaded document (v2.0 F6).
func (h *DocumentHandler) triggerOCR(d *document.Document) {
	if h.ocr == nil || !h.ocr.Enabled() || d.FilePath == "" {
		return
	}
	safe.Go("OCR dokumen peserta", func() {
		h.ocr.ProcessDocument(context.Background(), d.ID, d.ParticipantID, d.FilePath, d.DocumentType)
	})
}

// ListByParticipant godoc
// @Summary      Daftar dokumen per peserta (admin)
// @Tags         documents
// @Security     BearerAuth
// @Param        participant_id path string true "Participant ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/participants/{participant_id}/documents [get]
func (h *DocumentHandler) ListByParticipant(c echo.Context) error {
	docs, err := h.docs.ListByParticipant(c.Request().Context(), c.Param("participant_id"))
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(docs))
}

// ListAllDocuments godoc
// @Summary      Daftar dokumen global (admin) — filter status / participant
// @Tags         documents
// @Security     BearerAuth
// @Param        status query string false "Filter status (menunggu/disetujui/ditolak)"
// @Param        participant_id query string false "Filter peserta"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/documents [get]
func (h *DocumentHandler) ListAllDocuments(c echo.Context) error {
	ctx := c.Request().Context()
	f := document.Filter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryPageSize(c, "per_page", 20),
	}
	if s := c.QueryParam("status"); s != "" {
		f.Status = &s
	}
	if pid := c.QueryParam("participant_id"); pid != "" {
		f.ParticipantID = &pid
	}
	docs, total, err := h.docs.List(ctx, f)
	if err != nil {
		return serverErr(c, err)
	}

	response := pageResponse(docs, total, f.Page, f.PerPage)
	// The review page's "N of M approved" summary describes every document of
	// the participant being reviewed, not the page or the status filter in front
	// of it — counting the filtered rows made the figure agree with itself and
	// with nothing else.
	if f.ParticipantID != nil {
		if summary, err := h.reviewSummary(ctx, *f.ParticipantID); err == nil {
			response["summary"] = summary
		}
	}
	return c.JSON(http.StatusOK, response)
}

// reviewSummary counts a participant's documents by outcome, ignoring whatever
// filter the reviewer is looking through.
func (h *DocumentHandler) reviewSummary(ctx context.Context, participantID string) (map[string]int, error) {
	all, err := h.docs.ListByParticipant(ctx, participantID)
	if err != nil {
		return nil, err
	}
	summary := map[string]int{
		"total":                 len(all),
		document.StatusApproved: 0,
		document.StatusPending:  0,
		document.StatusRejected: 0,
	}
	for _, d := range all {
		summary[d.Status]++
	}
	return summary, nil
}

func (h *DocumentHandler) UploadDocument(c echo.Context) error {
	var d document.Document
	if err := bindJSON(c, &d); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	d.ParticipantID = c.Param("participant_id")
	if err := h.docs.Create(c.Request().Context(), &d); err != nil {
		return serverErr(c, err)
	}
	h.triggerOCR(&d) // v2.0 F6 — async OCR via self-hosted Tesseract
	return c.JSON(http.StatusCreated, ok(d))
}

// GetOCRResult godoc
// @Summary      Hasil OCR untuk sebuah dokumen (v2.0 F3)
// @Tags         documents
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/documents/{id}/ocr-result [get]
func (h *DocumentHandler) GetOCRResult(c echo.Context) error {
	if h.ocrRepo == nil {
		return notFound(c, "hasil OCR tidak tersedia")
	}
	res, err := h.ocrRepo.GetByDocument(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "hasil OCR belum tersedia")
	}
	return c.JSON(http.StatusOK, ok(res))
}

// ReviewDocument godoc
// @Summary      Setujui/tolak dokumen + kirim WA DOC_REJECTED (FR-PORTAL-05)
// @Tags         documents
// @Security     BearerAuth
// @Param        id path string true "Document ID"
// @Accept       json
// @Success      200 {object} map[string]interface{}
// @Router       /admin/documents/{id}/review [patch]
func (h *DocumentHandler) ReviewDocument(c echo.Context) error {
	var body struct {
		Status          string `json:"status" validate:"required,oneof=disetujui ditolak"`
		RejectionReason string `json:"rejection_reason" validate:"required_if=Status ditolak"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "status harus 'disetujui' atau 'ditolak'")
	}
	docID := c.Param("id")
	// The outcome is applied to the document itself (§14.4 Document.Approve /
	// Document.Reject) and the repository persists what that produced, so the
	// rules — a rejection says why, an approval drops the previous reason — live
	// in one place instead of being restated by every caller that reviews.
	reviewed := document.Document{ID: docID}
	if err := applyReview(&reviewed, body.Status, claimUserID(c), body.RejectionReason); err != nil {
		return badRequest(c, err.Error())
	}
	if err := h.docs.Review(c.Request().Context(), docID,
		reviewed.Status, *reviewed.ReviewedBy, reviewed.RejectionReason); err != nil {
		return serverErr(c, err)
	}

	// Async: send DOC_REJECTED WA when document is rejected
	if body.Status == document.StatusRejected && h.fonnte != nil && h.participants != nil {
		reason := body.RejectionReason
		safe.Go("notifikasi dokumen ditolak", func() {
			bgCtx := context.Background()
			// The document is reloaded for its participant id; the URL only
			// carries the document's own.
			doc, err := h.docs.GetByID(bgCtx, docID)
			if err != nil || doc == nil {
				return
			}
			p, err := h.participants.GetByID(bgCtx, doc.ParticipantID)
			if err != nil {
				return
			}
			portalBase := config.PortalBaseURL()
			_ = h.fonnte.SendDocRejected(bgCtx, p.Phone, p.Name,
				doc.DocumentType, reason, portalBase+"/portal/documents", p.ID)
			if h.email != nil && p.Email != "" {
				_ = h.email.SendEmailDocRejected(bgCtx, p.Email, p.Name,
					doc.DocumentType, reason, portalBase+"/portal/documents")
			}
		})
	}

	// §1.5/§1.6: when a document is approved, check whether the participant has
	// all documents approved (notify DOC_APPROVED) and whether the whole batch is
	// ready (auto-generate the airport checklist).
	if body.Status == document.StatusApproved {
		safe.Go("otomasi setelah dokumen disetujui", func() {
			h.onDocumentApproved(context.Background(), docID)
		})
	}

	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Dokumen berhasil direview"}))
}

// applyReview records the reviewer's decision on d. The status has already been
// checked against the schema's vocabulary by the validate tag; this maps it to
// the entity method that owns what the decision means.
func applyReview(d *document.Document, status, reviewerID, reason string) error {
	if status == document.StatusApproved {
		return d.Approve(reviewerID, time.Now())
	}
	return d.Reject(reviewerID, reason, time.Now())
}

// onDocumentApproved runs the post-approval automation (§1.5 + §1.6).
func (h *DocumentHandler) onDocumentApproved(ctx context.Context, docID string) {
	doc, err := h.docs.GetByID(ctx, docID)
	if err != nil || doc == nil {
		return
	}
	p, err := h.participants.GetByID(ctx, doc.ParticipantID)
	if err != nil {
		return
	}

	// Notify the participant once all their documents are approved.
	if h.allDocsApproved(ctx, p.ID) {
		if h.fonnte != nil {
			_ = h.fonnte.SendDocApproved(ctx, p.Phone, p.Name, p.ID)
		}
		if h.email != nil && p.Email != "" {
			_ = h.email.SendEmailDocApproved(ctx, p.Email, p.Name)
		}
	}

	// When every active participant in the batch has all documents approved, the
	// batch is ready — generate the airport checklist (idempotent via UNIQUE).
	if h.batchReady(ctx, p.BatchID) && h.airport != nil {
		if err := h.airport.InitForBatch(ctx, p.BatchID); err != nil {
			log.Printf("airport InitForBatch[%s] failed: %v", p.BatchID, err)
		}
	}
}

// allDocsApproved reports whether a participant has at least one document and all
// of their documents are approved (prompt §1.5 literal interpretation).
func (h *DocumentHandler) allDocsApproved(ctx context.Context, participantID string) bool {
	docs, err := h.docs.ListByParticipant(ctx, participantID)
	if err != nil || len(docs) == 0 {
		return false
	}
	for _, d := range docs {
		if d.Status != document.StatusApproved {
			return false
		}
	}
	return true
}

// batchReady reports whether every active participant in the batch has all
// documents approved.
func (h *DocumentHandler) batchReady(ctx context.Context, batchID string) bool {
	pts, err := h.participants.ListByBatch(ctx, batchID)
	if err != nil || len(pts) == 0 {
		return false
	}
	for _, pt := range pts {
		if !pt.IsActive {
			continue
		}
		if !h.allDocsApproved(ctx, pt.ID) {
			return false
		}
	}
	return true
}

func (h *DocumentHandler) DeleteDocument(c echo.Context) error {
	if err := h.docs.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return serverErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Country Requirements ──────────────────────────────────────────────────────

// ListCountryRequirements godoc
// @Summary      Daftar persyaratan dokumen per negara (publik)
// @Tags         country-requirements
// @Param        country_code path string true "Kode negara ISO (cth: JP)"
// @Success      200 {object} map[string]interface{}
// @Router       /country-requirements/{country_code} [get]
func (h *DocumentHandler) ListCountryRequirements(c echo.Context) error {
	reqs, err := h.reqs.List(c.Request().Context(), c.Param("country_code"))
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(reqs))
}

func (h *DocumentHandler) ListAllCountryRequirements(c echo.Context) error {
	countryCode := c.QueryParam("country_code")
	reqs, err := h.reqs.List(c.Request().Context(), countryCode)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(reqs))
}

// CreateCountryRequirement godoc
// @Summary      Tambah persyaratan dokumen baru per negara (admin) (US-ADM-07)
// @Tags         country-requirements
// @Security     BearerAuth
// @Accept       json
// @Success      201 {object} map[string]interface{}
// @Router       /admin/country-requirements [post]
func (h *DocumentHandler) CreateCountryRequirement(c echo.Context) error {
	var req document.CountryRequirement
	if err := bindJSON(c, &req); err != nil {
		return invalidPayload(c, err, "country_code dan document_type harus diisi")
	}
	if err := h.reqs.Create(c.Request().Context(), &req); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(req))
}

func (h *DocumentHandler) UpdateCountryRequirement(c echo.Context) error {
	var req document.CountryRequirement
	if err := bindJSON(c, &req); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	req.ID = c.Param("id")
	if err := h.reqs.Update(c.Request().Context(), &req); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(req))
}

func (h *DocumentHandler) DeleteCountryRequirement(c echo.Context) error {
	if err := h.reqs.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return serverErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
