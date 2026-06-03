package httpdelivery

import (
	"context"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

type DocumentHandler struct {
	docs         document.Repository
	reqs         document.CountryRequirementRepository
	participants participant.Repository
	fonnte       *service.FonnteService
}

func NewDocumentHandler(
	docs document.Repository,
	reqs document.CountryRequirementRepository,
	participants participant.Repository,
	fonnte *service.FonnteService,
) *DocumentHandler {
	return &DocumentHandler{docs: docs, reqs: reqs, participants: participants, fonnte: fonnte}
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

func (h *DocumentHandler) UploadDocument(c echo.Context) error {
	var d document.Document
	if err := bindJSON(c, &d); err != nil {
		return badRequest(c, "format tidak valid")
	}
	d.ParticipantID = c.Param("participant_id")
	if err := h.docs.Create(c.Request().Context(), &d); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(d))
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
		go func(reason string) {
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
		}(body.RejectionReason)
	}

	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Dokumen berhasil direview"}))
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
