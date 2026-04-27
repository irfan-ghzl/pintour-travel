package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

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
		customerEmail, customerPhone, validUntil    *string
		notes, pdfURL                               *string
		totalPrice                                  float64
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

// PrintQuotation godoc
//
//	@Summary     Print-friendly HTML quotation (admin)
//	@Tags        quotations
//	@Produce     html
//	@Security    BearerAuth
//	@Param       id path string true "Quotation ID"
//	@Success     200 {string} string "HTML"
//	@Router      /api/v1/admin/quotations/{id}/print [get]
func (h *QuotationHandler) PrintQuotation(c echo.Context) error {
	id := c.Param("id")

	var (
		qID, title, customerName, status, createdAt string
		customerEmail, customerPhone, validUntil    *string
		notes                                       *string
		totalPrice                                  float64
	)
	row := h.db.QueryRowContext(c.Request().Context(), `
		SELECT q.id, q.title, q.customer_name, q.customer_email, q.customer_phone,
		       q.valid_until, q.total_price, q.notes, q.status, q.created_at
		FROM quotations q WHERE q.id=$1`, id)
	if err := row.Scan(&qID, &title, &customerName, &customerEmail, &customerPhone,
		&validUntil, &totalPrice, &notes, &status, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "quotation not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotation")
	}

	itemRows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT description, category, quantity, unit_price, total_price
		FROM quotation_items WHERE quotation_id=$1 ORDER BY created_at ASC`, qID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch items")
	}
	defer itemRows.Close()

	type item struct {
		Description string
		Category    string
		Quantity    int
		UnitPrice   float64
		TotalPrice  float64
	}
	var items []item
	for itemRows.Next() {
		var it item
		var cat *string
		if err := itemRows.Scan(&it.Description, &cat, &it.Quantity, &it.UnitPrice, &it.TotalPrice); err != nil {
			continue
		}
		if cat != nil {
			it.Category = *cat
		}
		items = append(items, it)
	}

	safeStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	fmtRp := func(v float64) string {
		return fmt.Sprintf("Rp %s", formatNumber(v))
	}

	rowsHTML := ""
	for i, it := range items {
		rowsHTML += fmt.Sprintf(`<tr>
			<td>%d</td><td>%s</td><td>%s</td>
			<td class="num">%d</td>
			<td class="num">%s</td>
			<td class="num">%s</td>
		</tr>`, i+1, htmlEscape(it.Description), htmlEscape(it.Category),
			it.Quantity, fmtRp(it.UnitPrice), fmtRp(it.TotalPrice))
	}

	emailLine := ""
	if e := safeStr(customerEmail); e != "" {
		emailLine = fmt.Sprintf(`<p>Email: %s</p>`, htmlEscape(e))
	}
	phoneLine := ""
	if p := safeStr(customerPhone); p != "" {
		phoneLine = fmt.Sprintf(`<p>Telepon: %s</p>`, htmlEscape(p))
	}
	validLine := ""
	if v := safeStr(validUntil); v != "" {
		validLine = fmt.Sprintf(`<p><strong>Berlaku hingga:</strong> %s</p>`, htmlEscape(v[:10]))
	}
	notesLine := ""
	if n := safeStr(notes); n != "" {
		notesLine = fmt.Sprintf(`<div class="notes"><strong>Catatan:</strong><br>%s</div>`, htmlEscape(n))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<title>Penawaran: %s</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: Arial, sans-serif; font-size: 13px; color: #222; padding: 32px; }
  h1 { font-size: 22px; margin-bottom: 4px; }
  .subtitle { color: #666; margin-bottom: 24px; font-size: 12px; }
  .header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 28px; }
  .company { font-weight: bold; font-size: 16px; }
  .customer { background: #f5f5f5; padding: 12px 16px; border-radius: 6px; margin-bottom: 20px; }
  table { width: 100%%; border-collapse: collapse; margin-bottom: 20px; }
  th { background: #1e3a5f; color: white; text-align: left; padding: 8px 10px; font-size: 12px; }
  td { padding: 8px 10px; border-bottom: 1px solid #eee; }
  .num { text-align: right; }
  .total-row td { font-weight: bold; background: #f5f5f5; }
  .notes { background: #fffbe6; border-left: 3px solid #f59e0b; padding: 10px 14px; margin-bottom: 16px; border-radius: 4px; }
  .footer { margin-top: 40px; font-size: 11px; color: #888; border-top: 1px solid #eee; padding-top: 12px; }
  @media print { body { padding: 0; } }
</style>
</head>
<body>
<div class="header">
  <div>
    <div class="company">🗺️ Pintour Travel</div>
    <p style="color:#666;font-size:12px;">Tour & Travel Consultant</p>
  </div>
  <div style="text-align:right">
    <h1>PENAWARAN HARGA</h1>
    <div class="subtitle">No: %s | %s</div>
  </div>
</div>
<div class="customer">
  <strong>Kepada Yth:</strong>
  <p>%s</p>
  %s%s
</div>
%s%s
<table>
  <thead>
    <tr>
      <th>#</th><th>Deskripsi</th><th>Kategori</th>
      <th class="num">Qty</th><th class="num">Harga Satuan</th><th class="num">Total</th>
    </tr>
  </thead>
  <tbody>
    %s
    <tr class="total-row">
      <td colspan="5" class="num">TOTAL</td>
      <td class="num">%s</td>
    </tr>
  </tbody>
</table>
<div class="footer">
  <p>Dokumen ini dibuat secara otomatis oleh sistem Pintour Travel.</p>
  <p>Dicetak pada: %s</p>
</div>
<script>window.onload = function(){ window.print(); }</script>
</body>
</html>`,
		htmlEscape(title),
		htmlEscape(qID[:8]),
		createdAt[:10],
		htmlEscape(customerName),
		emailLine, phoneLine,
		validLine, notesLine,
		rowsHTML,
		fmtRp(totalPrice),
		createdAt[:10],
	)

	return c.HTML(http.StatusOK, html)
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func formatNumber(f float64) string {
	s := fmt.Sprintf("%.0f", f)
	result := ""
	n := len(s)
	for i, ch := range s {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(ch)
	}
	return result
}
