package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
)

type gatewayOrderRepo struct{ db dbtx }

func NewGatewayOrderRepo(db *sql.DB) invoice.GatewayOrderRepository {
	return &gatewayOrderRepo{db}
}

func (r *gatewayOrderRepo) Create(ctx context.Context, o *invoice.GatewayOrder) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO invoice_gateway_orders (id, invoice_id, order_id, snap_token, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		RETURNING id, created_at`,
		o.InvoiceID, o.OrderID, o.SnapToken,
	).Scan(&o.ID, &o.CreatedAt)
}

// FindInvoiceIDByOrder resolves an order back to its invoice.
//
// It falls back to the column the order id used to live in, so a session opened
// before this table existed — and paid afterwards — is still matched rather than
// reported as an unknown order.
func (r *gatewayOrderRepo) FindInvoiceIDByOrder(ctx context.Context, orderID string) (string, error) {
	var invoiceID string
	err := r.db.QueryRowContext(ctx, `
		SELECT invoice_id FROM invoice_gateway_orders WHERE order_id=$1
		UNION ALL
		SELECT id FROM invoices WHERE midtrans_order_id=$1 AND deleted_at IS NULL
		LIMIT 1`, orderID).Scan(&invoiceID)
	if err != nil {
		return "", err
	}
	return invoiceID, nil
}

// ClaimNotification inserts the notification's identity, reporting whether this
// call is the one that got there first.
//
// ON CONFLICT DO NOTHING rather than a read-then-write: two redeliveries
// arriving at once would both find the key absent and both go on to record a
// payment. The unique index is the arbiter, and it can only pick one.
func (r *gatewayOrderRepo) ClaimNotification(ctx context.Context, n invoice.GatewayNotification) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO gateway_notifications
			(idempotency_key, invoice_id, order_id, transaction_id, transaction_status)
		VALUES ($1, $2, $3, NULLIF($4,''), $5)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		n.IdempotencyKey(), n.InvoiceID, n.OrderID, n.TransactionID, n.TransactionStatus)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
