package httpdelivery

import (
	"fmt"
	"net/http"
	"strings"

	quotationsvc "github.com/irfan-ghzl/pintour-travel/internal/application/quotation"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/quotation"
	"github.com/labstack/echo/v4"
)

// QuotationHandler handles HTTP requests for quotations.
type QuotationHandler struct {
	svc *quotationsvc.QuotationService
}

// NewQuotationHandler creates a new QuotationHandler.
func NewQuotationHandler(svc *quotationsvc.QuotationService) *QuotationHandler {
	return &QuotationHandler{svc: svc}
}

// QuotationItemInput is a single line item in a CreateQuotation request.
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

	var totalPrice float64
	var items []quotation.CreateItemParams
	for _, it := range req.Items {
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		totalPrice += float64(qty) * it.UnitPrice
		items = append(items, quotation.CreateItemParams{
			Description: it.Description,
			Category:    it.Category,
			Quantity:    qty,
			UnitPrice:   it.UnitPrice,
		})
	}

	var inquiryID *string
	if req.InquiryID != "" {
		inquiryID = &req.InquiryID
	}
	var validUntil *string
	if req.ValidUntil != "" {
		validUntil = &req.ValidUntil
	}
	var createdByPtr *string
	if createdBy != "" {
		createdByPtr = &createdBy
	}

	id, err := h.svc.Create(c.Request().Context(), quotation.CreateParams{
		InquiryID:     inquiryID,
		CreatedBy:     createdByPtr,
		Title:         req.Title,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		ValidUntil:    validUntil,
		TotalPrice:    totalPrice,
		Notes:         req.Notes,
		Items:         items,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create quotation: "+err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id, "total_price": totalPrice})
}

// ListQuotations godoc
//
//	@Summary     List all quotations (admin)
//	@Tags        quotations
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/quotations [get]
func (h *QuotationHandler) ListQuotations(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 20)

	quotations, total, err := h.svc.List(c.Request().Context(), quotation.Filter{Page: page, PerPage: perPage})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotations")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": quotations, "total": total, "page": page, "per_page": perPage,
	})
}

// GetQuotation godoc
//
//	@Summary     Get a quotation (admin)
//	@Tags        quotations
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Quotation ID"
//	@Success     200 {object} quotation.Detail
//	@Router      /api/v1/admin/quotations/{id} [get]
func (h *QuotationHandler) GetQuotation(c echo.Context) error {
	detail, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotation")
	}
	if detail == nil {
		return echo.NewHTTPError(http.StatusNotFound, "quotation not found")
	}
	return c.JSON(http.StatusOK, detail)
}

// PrintQuotation godoc
//
//	@Summary     Print-friendly HTML quotation
//	@Tags        quotations
//	@Produce     html
//	@Param       id path string true "Quotation ID"
//	@Success     200 {string} string "HTML"
//	@Router      /api/v1/quotations/{id}/print [get]
func (h *QuotationHandler) PrintQuotation(c echo.Context) error {
	detail, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch quotation")
	}
	if detail == nil {
		return echo.NewHTTPError(http.StatusNotFound, "quotation not found")
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
	for i, it := range detail.Items {
		rowsHTML += fmt.Sprintf(`<tr>
			<td>%d</td><td>%s</td><td>%s</td>
			<td class="num">%d</td>
			<td class="num">%s</td>
			<td class="num">%s</td>
		</tr>`, i+1,
			htmlEscape(it.Description), htmlEscape(safeStr(it.Category)),
			it.Quantity, fmtRp(it.UnitPrice), fmtRp(it.TotalPrice))
	}

	emailLine := ""
	if e := safeStr(detail.CustomerEmail); e != "" {
		emailLine = fmt.Sprintf(`<p>Email: %s</p>`, htmlEscape(e))
	}
	phoneLine := ""
	if p := safeStr(detail.CustomerPhone); p != "" {
		phoneLine = fmt.Sprintf(`<p>Telepon: %s</p>`, htmlEscape(p))
	}
	validLine := ""
	if v := safeStr(detail.ValidUntil); v != "" {
		d := v
		if len(d) >= 10 {
			d = d[:10]
		}
		validLine = fmt.Sprintf(`<p><strong>Berlaku hingga:</strong> %s</p>`, htmlEscape(d))
	}
	notesLine := ""
	if n := safeStr(detail.Notes); n != "" {
		notesLine = fmt.Sprintf(`<div class="notes"><strong>Catatan:</strong><br>%s</div>`, htmlEscape(n))
	}

	dateStr := detail.CreatedAt
	if len(dateStr) >= 10 {
		dateStr = dateStr[:10]
	}
	shortID := detail.ID
	if len(shortID) >= 8 {
		shortID = shortID[:8]
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
    <p style="color:#666;font-size:12px;">Tour &amp; Travel Consultant</p>
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
		htmlEscape(detail.Title),
		htmlEscape(shortID), dateStr,
		htmlEscape(detail.CustomerName),
		emailLine, phoneLine,
		validLine, notesLine,
		rowsHTML,
		fmtRp(detail.TotalPrice),
		dateStr,
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
