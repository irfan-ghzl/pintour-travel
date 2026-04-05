package handler

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

// QuotationHandler handles HTTP requests for quotations.
type QuotationHandler struct {
	db *sql.DB
}

// NewQuotationHandler creates a new QuotationHandler.
func NewQuotationHandler(db *sql.DB) *QuotationHandler {
	return &QuotationHandler{db: db}
}

// QuotationItemInput represents a single line item in a quotation request.
type QuotationItemInput struct {
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

// CreateQuotationRequest is the body for POST /api/v1/admin/quotations.
type CreateQuotationRequest struct {
	InquiryID     string               `json:"inquiry_id"`
	Title         string               `json:"title"`
	CustomerName  string               `json:"customer_name"`
	CustomerEmail string               `json:"customer_email"`
	CustomerPhone string               `json:"customer_phone"`
	ValidUntil    string               `json:"valid_until"`
	Notes         string               `json:"notes"`
	Items         []QuotationItemInput `json:"items"`
}

// CreateQuotation godoc
//
//	@Summary     Create a quotation (admin)
//	@Tags        quotations
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       body body CreateQuotationRequest true "Quotation payload"
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/quotations [post]
func (h *QuotationHandler) CreateQuotation(c echo.Context) error {
	createdBy, _ := c.Get("user_id").(string)

	var req CreateQuotationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Title == "" || req.CustomerName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and customer_name are required")
	}

	// Calculate total price
	var totalPrice float64
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		totalPrice += float64(item.Quantity) * item.UnitPrice
	}

	tx, err := h.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start transaction")
	}
	defer tx.Rollback() //nolint:errcheck

	var quotationID string
	var inquiryID *string
	if req.InquiryID != "" {
		inquiryID = &req.InquiryID
	}
	var validUntil *string
	if req.ValidUntil != "" {
		validUntil = &req.ValidUntil
	}

	err = tx.QueryRowContext(c.Request().Context(), `
		INSERT INTO quotations
		  (inquiry_id, created_by, title, customer_name, customer_email,
		   customer_phone, valid_until, total_price, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		inquiryID, createdBy, req.Title, req.CustomerName, req.CustomerEmail,
		req.CustomerPhone, validUntil, totalPrice, req.Notes,
	).Scan(&quotationID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create quotation: "+err.Error())
	}

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		_, err := tx.ExecContext(c.Request().Context(), `
			INSERT INTO quotation_items (quotation_id, description, category, quantity, unit_price)
			VALUES ($1,$2,$3,$4,$5)`,
			quotationID, item.Description, item.Category, item.Quantity, item.UnitPrice,
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to add quotation item")
		}
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit transaction")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":          quotationID,
		"total_price": totalPrice,
	})
}

// GetQuotation godoc
//
//	@Summary     Get a quotation (admin)
//	@Tags        quotations
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Quotation ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/quotations/{id} [get]
func (h *QuotationHandler) GetQuotation(c echo.Context) error {
	id := c.Param("id")

	var (
		qID, title, customerName, status, createdAt string
		customerEmail, customerPhone, validUntil     *string
		notes, pdfURL                                *string
		totalPrice                                   float64
	)
	row := h.db.QueryRowContext(c.Request().Context(), `
		SELECT id, title, customer_name, customer_email, customer_phone,
		       valid_until, total_price, notes, status, pdf_url, created_at
		FROM quotations WHERE id=$1`, id)
	if err := row.Scan(&qID, &title, &customerName, &customerEmail, &customerPhone,
		&validUntil, &totalPrice, &notes, &status, &pdfURL, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "quotation not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotation")
	}

	itemRows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT id, description, category, quantity, unit_price, total_price
		FROM quotation_items WHERE quotation_id=$1`, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotation items")
	}
	defer itemRows.Close()

	var items []map[string]interface{}
	for itemRows.Next() {
		var itemID, desc string
		var category *string
		var qty int
		var unitPrice, itemTotal float64
		if err := itemRows.Scan(&itemID, &desc, &category, &qty, &unitPrice, &itemTotal); err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"id": itemID, "description": desc, "category": category,
			"quantity": qty, "unit_price": unitPrice, "total_price": itemTotal,
		})
	}
	if items == nil {
		items = []map[string]interface{}{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id": qID, "title": title, "customer_name": customerName,
		"customer_email": customerEmail, "customer_phone": customerPhone,
		"valid_until": validUntil, "total_price": totalPrice,
		"notes": notes, "status": status, "pdf_url": pdfURL,
		"created_at": createdAt, "items": items,
	})
}

// ListQuotations godoc
//
//	@Summary     List all quotations (admin)
//	@Tags        quotations
//	@Produce     json
//	@Security    BearerAuth
//	@Param       page     query int false "Page"
//	@Param       per_page query int false "Per page"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/quotations [get]
func (h *QuotationHandler) ListQuotations(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 20)
	offset := (page - 1) * perPage

	rows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT q.id, q.title, q.customer_name, q.total_price,
		       q.status, q.created_at, COALESCE(u.name,'') AS created_by_name
		FROM quotations q
		LEFT JOIN users u ON u.id = q.created_by
		ORDER BY q.created_at DESC
		LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotations")
	}
	defer rows.Close()

	var quotations []map[string]interface{}
	for rows.Next() {
		var id, title, customerName, status, createdAt, createdByName string
		var totalPrice float64
		if err := rows.Scan(&id, &title, &customerName, &totalPrice, &status, &createdAt, &createdByName); err != nil {
			continue
		}
		quotations = append(quotations, map[string]interface{}{
			"id": id, "title": title, "customer_name": customerName,
			"total_price": totalPrice, "status": status,
			"created_at": createdAt, "created_by_name": createdByName,
		})
	}
	if quotations == nil {
		quotations = []map[string]interface{}{}
	}

	var total int
	h.db.QueryRowContext(c.Request().Context(), `SELECT COUNT(*) FROM quotations`).Scan(&total)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": quotations, "total": total, "page": page, "per_page": perPage,
	})
}
