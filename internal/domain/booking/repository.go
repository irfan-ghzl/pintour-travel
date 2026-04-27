package booking

import "context"

// Repository is the persistence abstraction for the booking domain.
type Repository interface {
	List(ctx context.Context, f Filter) ([]Booking, int, error)
	GetByID(ctx context.Context, id string) (*Detail, error)
	Create(ctx context.Context, p CreateParams) (id, code string, err error)
	UpdatePaymentStatus(ctx context.Context, id, status string) error
	UpdateBookingStatus(ctx context.Context, id, status string) error
	Delete(ctx context.Context, id string) error
}
