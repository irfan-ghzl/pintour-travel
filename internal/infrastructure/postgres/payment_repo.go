package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/payment"
)

// PaymentRepo implements payment.Repository using PostgreSQL.
type PaymentRepo struct {
	db *sql.DB
}

// NewPaymentRepo creates a new PaymentRepo.
func NewPaymentRepo(db *sql.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) ListByBooking(ctx context.Context, bookingID string) ([]payment.Payment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, booking_id, payment_type, amount, paid_at, proof_url, notes,
		       verified_by, verified_at, created_at
		FROM payments
		WHERE booking_id = $1
		ORDER BY paid_at ASC`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []payment.Payment
	for rows.Next() {
		var p payment.Payment
		var paidAt, createdAt time.Time
		var verifiedAt *time.Time
		if err := rows.Scan(
			&p.ID, &p.BookingID, &p.PaymentType, &p.Amount, &paidAt,
			&p.ProofURL, &p.Notes, &p.VerifiedBy, &verifiedAt, &createdAt,
		); err != nil {
			return nil, err
		}
		p.PaidAt = paidAt.Format(time.RFC3339)
		p.CreatedAt = createdAt.Format(time.RFC3339)
		if verifiedAt != nil {
			s := verifiedAt.Format(time.RFC3339)
			p.VerifiedAt = &s
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PaymentRepo) Create(ctx context.Context, p payment.CreateParams) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO payments (booking_id, payment_type, amount, paid_at, proof_url, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		p.BookingID, p.PaymentType, p.Amount, p.PaidAt, p.ProofURL, p.Notes,
	).Scan(&id)
	return id, err
}

func (r *PaymentRepo) Verify(ctx context.Context, id, verifiedByUserID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE payments SET verified_by=$2, verified_at=NOW() WHERE id=$1`, id, verifiedByUserID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PaymentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM payments WHERE id=$1`, id)
	return err
}
