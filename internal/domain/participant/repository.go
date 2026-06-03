package participant

import "context"

type Repository interface {
	Create(ctx context.Context, p *Participant) error
	Update(ctx context.Context, p *Participant) error
	GetByID(ctx context.Context, id string) (*Participant, error)
	GetByPhone(ctx context.Context, phone string) (*Participant, error)
	List(ctx context.Context, f Filter) ([]Participant, int, error)
	Activate(ctx context.Context, id string) error
	ListByBatch(ctx context.Context, batchID string) ([]Participant, error)
	// For scheduler: list participants departing in N days
	ListByDepartureDaysAhead(ctx context.Context, days int) ([]Participant, error)
	// For scheduler: list participants with unpaid invoices older than N days
	ListWithUnpaidInvoiceDaysOld(ctx context.Context, days int) ([]Participant, error)
}
