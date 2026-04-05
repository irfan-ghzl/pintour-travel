package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
)

// InquiryHandler handles HTTP requests for customer inquiries / leads.
type InquiryHandler struct {
	db              *sql.DB
	consultantPhone string
}

// NewInquiryHandler creates a new InquiryHandler.
func NewInquiryHandler(db *sql.DB, consultantPhone string) *InquiryHandler {
	return &InquiryHandler{db: db, consultantPhone: consultantPhone}
}

// CreateInquiry godoc
//
//	@Summary     Submit a consultation request (Build My Trip)
//	@Description Saves the customer inquiry and returns a pre-filled WhatsApp link
//	@Tags        inquiries
//	@Accept      json
//	@Produce     json
//	@Param       body body CreateInquiryRequest true "Inquiry payload"
//	@Success     201 {object} CreateInquiryResponse
//	@Router      /api/v1/inquiries [post]
func (h *InquiryHandler) CreateInquiry(c echo.Context) error {
	var req CreateInquiryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.FullName) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "full_name is required")
	}

	// Resolve package title for WA message
	var packageTitle string
	if req.TourPackageID != "" {
		h.db.QueryRowContext(c.Request().Context(),
			`SELECT title FROM tour_packages WHERE id = $1`, req.TourPackageID,
		).Scan(&packageTitle)
	}

	waLink := service.BuildWhatsAppLink(h.consultantPhone, service.WhatsAppLinkParams{
		FullName:      req.FullName,
		Destination:   req.Destination,
		NumPeople:     req.NumPeople,
		DurationDays:  req.DurationDays,
		DepartureDate: req.DepartureDate,
		Budget:        req.Budget,
		Notes:         req.Notes,
		PackageTitle:  packageTitle,
	})

	var (
		id        string
		createdAt string
	)

	var tourPkgID *string
	if req.TourPackageID != "" {
		tourPkgID = &req.TourPackageID
	}

	err := h.db.QueryRowContext(c.Request().Context(), `
		INSERT INTO inquiries
		  (full_name, email, phone, destination, tour_package_id,
		   num_people, budget, duration_days, departure_date, notes, wa_link)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at`,
		req.FullName, req.Email, req.Phone, req.Destination, tourPkgID,
		req.NumPeople, req.Budget, req.DurationDays, req.DepartureDate,
		req.Notes, waLink,
	).Scan(&id, &createdAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save inquiry: "+err.Error())
	}

	return c.JSON(http.StatusCreated, CreateInquiryResponse{
		ID:        id,
		WALink:    waLink,
		CreatedAt: createdAt,
		Message:   fmt.Sprintf("Terima kasih %s! Klik link WhatsApp untuk melanjutkan konsultasi.", req.FullName),
	})
}

// ListInquiries godoc
//
//	@Summary     List all inquiries (admin)
//	@Tags        inquiries
//	@Produce     json
//	@Security    BearerAuth
//	@Param       page     query int false "Page"
//	@Param       per_page query int false "Per page"
//	@Param       status   query string false "Filter by status"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/inquiries [get]
func (h *InquiryHandler) ListInquiries(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 20)
	status := c.QueryParam("status")
	offset := (page - 1) * perPage

	query := `
		SELECT i.id, i.full_name, i.email, i.phone, i.destination,
		       i.num_people, i.budget, i.duration_days, i.departure_date,
		       i.status, i.wa_link, i.created_at,
		       COALESCE(tp.title, '') AS package_title
		FROM inquiries i
		LEFT JOIN tour_packages tp ON tp.id = i.tour_package_id`

	args := []interface{}{}
	if status != "" {
		query += " WHERE i.status = $3"
		args = append(args, status)
		query += " ORDER BY i.created_at DESC LIMIT $1 OFFSET $2"
		args = append([]interface{}{perPage, offset}, args...)
	} else {
		query += " ORDER BY i.created_at DESC LIMIT $1 OFFSET $2"
		args = []interface{}{perPage, offset}
	}

	rows, err := h.db.QueryContext(c.Request().Context(), query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch inquiries")
	}
	defer rows.Close()

	var inquiries []map[string]interface{}
	for rows.Next() {
		var (
			id, fullName, statusVal, waLink, createdAt, packageTitle string
			email, phone, destination, departureDate                  *string
			numPeople                                                  int
			budget                                                     *float64
			durationDays                                               *int
		)
		if err := rows.Scan(
			&id, &fullName, &email, &phone, &destination,
			&numPeople, &budget, &durationDays, &departureDate,
			&statusVal, &waLink, &createdAt, &packageTitle,
		); err != nil {
			continue
		}
		inquiries = append(inquiries, map[string]interface{}{
			"id": id, "full_name": fullName, "email": email, "phone": phone,
			"destination": destination, "num_people": numPeople,
			"budget": budget, "duration_days": durationDays,
			"departure_date": departureDate, "status": statusVal,
			"wa_link": waLink, "created_at": createdAt,
			"package_title": packageTitle,
		})
	}
	if inquiries == nil {
		inquiries = []map[string]interface{}{}
	}

	var total int
	h.db.QueryRowContext(c.Request().Context(), `SELECT COUNT(*) FROM inquiries`).Scan(&total)

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
//	@Param       id   path string true "Inquiry ID"
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

	validStatuses := map[string]bool{
		"new": true, "contacted": true, "in_progress": true,
		"quoted": true, "booked": true, "closed": true,
	}
	if !validStatuses[body.Status] {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status value")
	}

	_, err := h.db.ExecContext(c.Request().Context(),
		`UPDATE inquiries SET status=$2, updated_at=NOW() WHERE id=$1`, id, body.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update status")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "status": body.Status})
}

// CreateInquiryRequest is the JSON body for POST /api/v1/inquiries.
type CreateInquiryRequest struct {
	FullName      string  `json:"full_name"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	Destination   string  `json:"destination"`
	TourPackageID string  `json:"tour_package_id"`
	NumPeople     int     `json:"num_people"`
	Budget        float64 `json:"budget"`
	DurationDays  int     `json:"duration_days"`
	DepartureDate string  `json:"departure_date"`
	Notes         string  `json:"notes"`
}

// CreateInquiryResponse is returned on successful inquiry creation.
type CreateInquiryResponse struct {
	ID        string `json:"id"`
	WALink    string `json:"wa_link"`
	CreatedAt string `json:"created_at"`
	Message   string `json:"message"`
}
