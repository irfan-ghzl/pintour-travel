package payment

import "context"

// Payment represents a single payment entry for a booking.
type Payment struct {
	ID          string  `json:"id"`
	BookingID   string  `json:"booking_id"`
	PaymentType string  `json:"payment_type"` // dp, pelunasan, full
	Amount      float64 `json:"amount"`
	PaidAt      string  `json:"paid_at"`
	ProofURL    *string `json:"proof_url,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	VerifiedBy  *string `json:"verified_by,omitempty"`
	VerifiedAt  *string `json:"verified_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// CreateParams contains the data needed to record a new payment.
type CreateParams struct {
	BookingID   string
	PaymentType string
	Amount      float64
	PaidAt      string
	ProofURL    *string
	Notes       *string
}

// Repository is the persistence abstraction for the payment domain.
type Repository interface {
	ListByBooking(ctx context.Context, bookingID string) ([]Payment, error)
	Create(ctx context.Context, p CreateParams) (string, error)
	Verify(ctx context.Context, id, verifiedByUserID string) error
	Delete(ctx context.Context, id string) error
}
