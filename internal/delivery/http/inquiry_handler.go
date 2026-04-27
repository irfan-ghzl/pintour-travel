package httpdelivery

import (
	"net/http"
	"strings"

	inquirysvc "github.com/irfan-ghzl/pintour-travel/internal/application/inquiry"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/inquiry"
	"github.com/labstack/echo/v4"
)

// InquiryHandler handles HTTP requests for customer inquiries.
type InquiryHandler struct {
	svc *inquirysvc.InquiryService
}

// NewInquiryHandler creates a new InquiryHandler.
func NewInquiryHandler(svc *inquirysvc.InquiryService) *InquiryHandler {
	return &InquiryHandler{svc: svc}
}

// createInquiryRequest is the body for POST /api/v1/inquiries.
type createInquiryRequest struct {
	FullName      string  `json:"full_name"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	Destination   string  `json:"destination"`
	TourPackageID string  `json:"tour_package_id"`
	NumPeople     int     `json:"num_people"`
	Budget        float64 `json:"budget"`
	DurationDays  int     `json:"duration_days"`
	DepartureDate *string `json:"departure_date"`
	Notes         string  `json:"notes"`
}

// CreateInquiry godoc
//
//	@Summary     Submit a consultation request (Build My Trip)
//	@Tags        inquiries
//	@Accept      json
//	@Produce     json
//	@Param       body body createInquiryRequest true "Inquiry payload"
//	@Success     201 {object} inquirysvc.SubmitResult
//	@Router      /api/v1/inquiries [post]
func (h *InquiryHandler) CreateInquiry(c echo.Context) error {
	var req createInquiryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.FullName) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "full_name is required")
	}

	result, err := h.svc.Submit(c.Request().Context(), inquirysvc.SubmitInquiryInput{
		FullName:      req.FullName,
		Email:         req.Email,
		Phone:         req.Phone,
		Destination:   req.Destination,
		TourPackageID: req.TourPackageID,
		NumPeople:     req.NumPeople,
		Budget:        req.Budget,
		DurationDays:  req.DurationDays,
		DepartureDate: req.DepartureDate,
		Notes:         req.Notes,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save inquiry: "+err.Error())
	}
	return c.JSON(http.StatusCreated, result)
}

// ListInquiries godoc
//
//	@Summary     List all inquiries (admin)
//	@Tags        inquiries
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/inquiries [get]
func (h *InquiryHandler) ListInquiries(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 20)

	inquiries, total, err := h.svc.ListInquiries(c.Request().Context(), inquiry.Filter{
		Status:  c.QueryParam("status"),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch inquiries")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": inquiries, "total": total, "page": page, "per_page": perPage,
	})
}

// UpdateInquiryStatus godoc
//
//	@Summary     Update inquiry status (admin)
//	@Tags        inquiries
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Inquiry ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/inquiries/{id}/status [patch]
func (h *InquiryHandler) UpdateInquiryStatus(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&body); err != nil || body.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "status is required")
	}
	if !inquirysvc.ValidStatuses[body.Status] {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status value")
	}

	if err := h.svc.UpdateStatus(c.Request().Context(), id, body.Status); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update status")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "status": body.Status})
}
