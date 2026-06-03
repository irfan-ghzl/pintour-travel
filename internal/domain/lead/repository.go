package lead

import "context"

type Repository interface {
	Create(ctx context.Context, l *Lead) error
	Update(ctx context.Context, l *Lead) error
	GetByID(ctx context.Context, id string) (*Lead, error)
	List(ctx context.Context, f Filter) ([]Lead, int, error)
	UpdateStatus(ctx context.Context, id, status, changedBy string) error
	AssignTo(ctx context.Context, leadID, consultantID string) error
	MarkConverted(ctx context.Context, leadID string) error
	CountActiveByConsultant(ctx context.Context, consultantID string) (int, error)
}

type NoteRepository interface {
	Create(ctx context.Context, n *Note) error
	ListByLead(ctx context.Context, leadID string) ([]Note, error)
}
