package invoice

import (
	"context"
	"strings"
	"time"
)

// GatewayOrder is one payment session opened against an invoice.
//
// A participant who opens the payment page, reloads, and then pays in the first
// tab sends a notification carrying the FIRST session's order id. While an
// invoice remembered only its latest order, that notification matched nothing
// and the money was received by the gateway but never recorded. So sessions
// accumulate rather than overwrite.
type GatewayOrder struct {
	ID        string    `json:"id"`
	InvoiceID string    `json:"invoice_id"`
	OrderID   string    `json:"order_id"`
	SnapToken string    `json:"snap_token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// GatewayNotification is a notification from the payment gateway, identified by
// the transaction it reports rather than by the state of the invoice.
type GatewayNotification struct {
	InvoiceID         string
	OrderID           string
	TransactionID     string
	TransactionStatus string
}

// IdempotencyKey identifies the transaction this notification reports, which is
// what repeated delivery has to be recognised by.
//
// Keying on invoice status instead is what let the same down payment be counted
// five times: a partially paid invoice is still "menunggu_bayar", so nothing
// about it said the money had already been seen. The transaction id is the
// gateway's own identifier for the payment; when it is absent the status stands
// in, so a redelivered settlement for one order still collides with itself.
func (n GatewayNotification) IdempotencyKey() string {
	discriminator := n.TransactionID
	if discriminator == "" {
		discriminator = n.TransactionStatus
	}
	return strings.Join([]string{n.OrderID, discriminator}, ":")
}

// GatewayOrderRepository stores the sessions opened against an invoice and
// resolves a notification's order back to the invoice it belongs to.
type GatewayOrderRepository interface {
	Create(ctx context.Context, o *GatewayOrder) error
	// FindInvoiceIDByOrder returns the invoice a gateway order belongs to.
	FindInvoiceIDByOrder(ctx context.Context, orderID string) (string, error)
	// ClaimNotification records that a notification has been applied. It reports
	// false — without an error — when the key was already present, which is how
	// a redelivery is told apart from a first arrival.
	ClaimNotification(ctx context.Context, n GatewayNotification) (claimed bool, err error)
}
