package httpdelivery

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

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
	ocr          *service.OCRService
	ocrRepo      document.OCRResultRepository
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
	var f document.Filter
	if s := c.QueryParam("status"); s != "" {
		f.Status = &s
	}
	if pid := c.QueryParam("participant_id"); pid != "" {
		f.ParticipantID = &pid
	}
	docs, err := h.docs.List(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(docs))
}

func (h *DocumentHandler) UploadDocument(c echo.Context) error {
	var d document.Document
	if err := bindJSON(c, &d); err != nil {
		return badRequest(c, "format tidak valid")
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
		Status          string `json:"status"`
		RejectionReason string `json:"rejection_reason"`
	}
	if err := bindJSON(c, &body); err != nil {
		return badRequest(c, "format tidak valid")
	}
	if body.Status != "disetujui" && body.Status != "ditolak" {
		return badRequest(c, "status harus 'disetujui' atau 'ditolak'")
	}
	if body.Status == "ditolak" && body.RejectionReason == "" {
		return badRequest(c, "alasan penolakan harus diisi saat menolak dokumen")
	}
	docID := c.Param("id")
	if err := h.docs.Review(c.Request().Context(), docID, body.Status, claimUserID(c), body.RejectionReason); err != nil {
		return serverErr(c, err)
	}

	// Async: send DOC_REJECTED WA when document is rejected
	if body.Status == "ditolak" && h.fonnte != nil && h.participants != nil {
		reason := body.RejectionReason
		safe.Go("notifikasi dokumen ditolak", func() {
			bgCtx := context.Background()
			// Reload doc to get participant_id (we know it from URL param, but need participant data)
			docs, err := h.docs.ListByParticipant(bgCtx, "")
			_ = docs
			_ = err
			// Easier path: lookup the document directly to get participant
			doc, err := h.docs.GetByID(bgCtx, docID)
			if err != nil || doc == nil {
				return
			}
			p, err := h.participants.GetByID(bgCtx, doc.ParticipantID)
			if err != nil {
				return
			}
			portalBase := os.Getenv("PORTAL_BASE_URL")
			if portalBase == "" {
				portalBase = "http://localhost:3000"
			}
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
	if body.Status == "disetujui" {
		safe.Go("otomasi setelah dokumen disetujui", func() {
			h.onDocumentApproved(context.Background(), docID)
		})
	}

	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Dokumen berhasil direview"}))
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
		if d.Status != "disetujui" {
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
		return badRequest(c, "format tidak valid")
	}
	if req.CountryCode == "" || req.DocumentType == "" {
		return badRequest(c, "country_code dan document_type harus diisi")
	}
	if err := h.reqs.Create(c.Request().Context(), &req); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(req))
}

func (h *DocumentHandler) UpdateCountryRequirement(c echo.Context) error {
	var req document.CountryRequirement
	if err := bindJSON(c, &req); err != nil {
		return badRequest(c, "format tidak valid")
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
