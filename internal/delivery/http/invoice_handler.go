package httpdelivery

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
)

type InvoiceHandler struct{ svc *invoicesvc.Service }

func NewInvoiceHandler(svc *invoicesvc.Service) *InvoiceHandler { return &InvoiceHandler{svc} }

// ListInvoices godoc
// @Summary      Daftar invoice
// @Tags         invoices
// @Security     BearerAuth
// @Param        status query string false "Filter status (diterbitkan/menunggu_bayar/dibayar/lunas)"
// @Param        participant_id query string false "Participant ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/invoices [get]
func (h *InvoiceHandler) ListInvoices(c echo.Context) error {
	f := domainInvoice.Filter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryPageSize(c, "per_page", 20),
	}
	if v := c.QueryParam("status"); v != "" { f.Status = &v }
	if v := c.QueryParam("participant_id"); v != "" { f.ParticipantID = &v }
	list, total, err := h.svc.ListInvoices(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, pageResponse(list, total, f.Page, f.PerPage))
}

// GetInvoice godoc
// @Summary      Detail invoice + payment proofs (AC-INV-02)
// @Tags         invoices
// @Security     BearerAuth
// @Param        id path string true "Invoice ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/invoices/{id} [get]
func (h *InvoiceHandler) GetInvoice(c echo.Context) error {
	inv, err := h.svc.GetInvoice(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "invoice tidak ditemukan")
	}
	proofs, _ := h.svc.ListProofs(c.Request().Context(), inv.ID)
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"invoice": inv, "payment_proofs": proofs,
	}))
}

// GetInvoicePDF godoc
// @Summary      Unduh invoice sebagai PDF
// @Tags         invoices
// @Security     BearerAuth
// @Param        id path string true "Invoice ID"
// @Produce      application/pdf
// @Success      200 {file} binary
// @Router       /admin/invoices/{id}/pdf [get]
func (h *InvoiceHandler) GetInvoicePDF(c echo.Context) error {
	pdfBytes, err := h.svc.GeneratePDF(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "invoice tidak ditemukan")
	}
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=invoice.pdf")
	return c.Blob(http.StatusOK, "application/pdf", pdfBytes)
}

// CreateInvoice godoc
// @Summary      Terbitkan invoice baru
// @Tags         invoices
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      201 {object} map[string]interface{}
// @Router       /admin/invoices [post]
func (h *InvoiceHandler) CreateInvoice(c echo.Context) error {
	var inv domainInvoice.Invoice
	if err := bindJSON(c, &inv); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	inv.IssuedBy = claimUserID(c)
	if err := h.svc.Create(c.Request().Context(), &inv); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(inv))
}

// ConfirmPayment godoc
// @Summary      Konfirmasi pembayaran lunas + aktivasi portal peserta
// @Tags         invoices
// @Security     BearerAuth
// @Param        id path string true "Invoice ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/invoices/{id}/confirm [post]
func (h *InvoiceHandler) ConfirmPayment(c echo.Context) error {
	portalBase := os.Getenv("PORTAL_BASE_URL")
	if portalBase == "" {
		portalBase = "http://localhost:3000"
	}
	if err := h.svc.ConfirmPayment(c.Request().Context(), c.Param("id"), claimUserID(c), portalBase); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]string{
		"message": "Pembayaran dikonfirmasi, portal peserta diaktifkan dan notifikasi WA terkirim",
	}))
}

// UploadProof godoc
// @Summary      Upload bukti transfer manual (admin) — alternatif jika peserta tidak bisa
// @Tags         invoices
// @Security     BearerAuth
// @Param        id path string true "Invoice ID"
// @Accept       json
// @Success      201 {object} map[string]interface{}
// @Router       /admin/invoices/{id}/proofs [post]
func (h *InvoiceHandler) UploadProof(c echo.Context) error {
	var proof domainInvoice.PaymentProof
	if err := bindJSON(c, &proof); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	proof.InvoiceID = c.Param("id")
	if err := h.svc.UploadProof(c.Request().Context(), &proof); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(proof))
}

// ReviewProof godoc
// @Summary      Setujui/tolak bukti transfer (AC-INV-02)
// @Tags         invoices
// @Security     BearerAuth
// @Param        id path string true "Invoice ID"
// @Param        proof_id path string true "Payment Proof ID"
// @Accept       json
// @Success      200 {object} map[string]interface{}
// @Router       /admin/invoices/{id}/proofs/{proof_id}/review [patch]
func (h *InvoiceHandler) ReviewProof(c echo.Context) error {
	var body struct {
		Status string `json:"status" validate:"required,oneof=disetujui ditolak"`
		Notes  string `json:"notes" validate:"required_if=Status ditolak"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "status harus 'disetujui' atau 'ditolak'")
	}
	portalBase := os.Getenv("PORTAL_BASE_URL")
	if portalBase == "" {
		portalBase = "http://localhost:3000"
	}
	// §1.4 + §1.3: settle payment (derive paid amount, activate portal when lunas)
	// and notify the participant.
	if err := h.svc.ReviewProofAndSettle(c.Request().Context(), c.Param("id"), c.Param("proof_id"),
		body.Status, claimUserID(c), body.Notes, portalBase); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(nil))
}
