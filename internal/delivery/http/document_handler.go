package httpdelivery

import (
	"net/http"

	documentsvc "github.com/irfan-ghzl/pintour-travel/internal/application/document"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/labstack/echo/v4"
)

// DocumentHandler handles HTTP requests for participant document management.
type DocumentHandler struct {
	svc *documentsvc.DocumentService
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(svc *documentsvc.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// ListDocumentsByBooking godoc
//
//	@Summary     List all participant documents for a booking (admin)
//	@Tags        documents
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     200 {array} document.Document
//	@Router      /api/v1/admin/bookings/{id}/documents [get]
func (h *DocumentHandler) ListDocumentsByBooking(c echo.Context) error {
	docs, err := h.svc.ListByBooking(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch documents")
	}
	if docs == nil {
		docs = []document.Document{}
	}
	return c.JSON(http.StatusOK, docs)
}

// ListDocumentsByParticipant godoc
//
//	@Summary     List documents for a specific participant (admin)
//	@Tags        documents
//	@Security    BearerAuth
//	@Param       pid path string true "Participant ID"
//	@Success     200 {array} document.Document
//	@Router      /api/v1/admin/participants/{pid}/documents [get]
func (h *DocumentHandler) ListDocumentsByParticipant(c echo.Context) error {
	docs, err := h.svc.ListByParticipant(c.Request().Context(), c.Param("pid"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch documents")
	}
	if docs == nil {
		docs = []document.Document{}
	}
	return c.JSON(http.StatusOK, docs)
}

// CreateDocument godoc
//
//	@Summary     Upload a document for a participant (admin)
//	@Tags        documents
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       pid path string true "Participant ID"
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/participants/{pid}/documents [post]
func (h *DocumentHandler) CreateDocument(c echo.Context) error {
	pid := c.Param("pid")
	var body struct {
		DocType string `json:"doc_type"`
		FileURL string `json:"file_url"`
		Notes   string `json:"notes"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if !documentsvc.ValidDocTypes[body.DocType] {
		return echo.NewHTTPError(http.StatusBadRequest, "doc_type must be one of: passport, ktp, bank_statement, visa_support, photo, other")
	}
	if body.FileURL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "file_url is required")
	}

	params := document.CreateParams{
		ParticipantID: pid,
		DocType:       body.DocType,
		FileURL:       body.FileURL,
	}
	if body.Notes != "" {
		params.Notes = &body.Notes
	}

	id, err := h.svc.CreateDocument(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create document")
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id})
}

// VerifyDocument godoc
//
//	@Summary     Mark a document as verified (admin)
//	@Tags        documents
//	@Security    BearerAuth
//	@Param       did path string true "Document ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/documents/{did}/verify [patch]
func (h *DocumentHandler) VerifyDocument(c echo.Context) error {
	did := c.Param("did")
	claims := c.Get("claims").(*auth.Claims)
	if err := h.svc.VerifyDocument(c.Request().Context(), did, claims.UserID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify document")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": did, "verified": true})
}

// DeleteDocument godoc
//
//	@Summary     Delete a document (admin)
//	@Tags        documents
//	@Security    BearerAuth
//	@Param       did path string true "Document ID"
//	@Success     204
//	@Router      /api/v1/admin/documents/{did} [delete]
func (h *DocumentHandler) DeleteDocument(c echo.Context) error {
	if err := h.svc.DeleteDocument(c.Request().Context(), c.Param("did")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete document")
	}
	return c.NoContent(http.StatusNoContent)
}
