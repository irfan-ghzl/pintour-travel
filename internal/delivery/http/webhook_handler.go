package httpdelivery

import (
	"errors"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// WebhookHandler handles public (non-JWT) webhooks from external providers.
type WebhookHandler struct {
	midtrans *service.MidtransService
	invoices *invoicesvc.Service
}

func NewWebhookHandler(midtrans *service.MidtransService, invoices *invoicesvc.Service) *WebhookHandler {
	return &WebhookHandler{midtrans: midtrans, invoices: invoices}
}

// midtransNotification is the subset of the Midtrans webhook body we use.
type midtransNotification struct {
	OrderID           string `json:"order_id"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	GrossAmount       string `json:"gross_amount"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
}

// HandleMidtrans processes the Midtrans payment notification (v2.0 F1).
//
//	@Summary  Webhook pembayaran Midtrans (publik, verifikasi signature SHA512)
//	@Tags     webhooks
//	@Accept   json
//	@Success  200 {object} map[string]interface{}
//	@Router   /webhooks/midtrans [post]
func (h *WebhookHandler) HandleMidtrans(c echo.Context) error {
	var n midtransNotification
	if err := bindJSON(c, &n); err != nil {
		return invalidPayload(c, err, "payload tidak valid")
	}

	// Always verify the signature before trusting anything in the body.
	if !h.midtrans.VerifySignature(n.OrderID, n.StatusCode, n.GrossAmount, n.SignatureKey) {
		log.Printf("midtrans webhook: invalid signature for order %q", n.OrderID)
		return c.JSON(http.StatusBadRequest, errResponse("INVALID_SIGNATURE", "signature webhook tidak valid"))
	}

	err := h.invoices.HandleGatewayNotification(c.Request().Context(),
		domainInvoice.GatewayNotification{
			OrderID:           n.OrderID,
			TransactionID:     n.TransactionID,
			TransactionStatus: n.TransactionStatus,
		}, n.PaymentType, n.GrossAmount, n.FraudStatus)
	if err != nil {
		if errors.Is(err, invoicesvc.ErrNotOwned) {
			// No such order. Final, and nothing a retry would change.
			return c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "invoice tidak ditemukan"))
		}
		// Anything else is our side failing, not the gateway's message being
		// wrong. This used to answer 200, which tells Midtrans the payment was
		// delivered and stops the retries — the one response that loses the
		// payment for good. A 500 keeps them coming until it works.
		log.Printf("midtrans webhook process order %q: %v", n.OrderID, err)
		return c.JSON(http.StatusInternalServerError,
			errResponse("PROCESSING_FAILED", "gagal memproses notifikasi, silakan kirim ulang"))
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"status": "ok"}))
}
