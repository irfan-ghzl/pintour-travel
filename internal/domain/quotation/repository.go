package quotation

import "context"

// Repository is the persistence abstraction for the quotation domain.
type Repository interface {
	Create(ctx context.Context, p CreateParams) (string, error)
	GetByID(ctx context.Context, id string) (*Detail, error)
	List(ctx context.Context, f Filter) ([]Quotation, int, error)
}
