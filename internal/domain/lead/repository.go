package lead

import (
	"context"
	"errors"
)

// ErrNotConvertible is returned by MarkConverted when the lead is not in a state
// it can be converted from — most often because it has been converted already.
// A conversion is at most one participant per lead, and this is what says so
// when two requests race past the caller's own status check.
var ErrNotConvertible = errors.New("lead tidak dalam status 'deal'")

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
