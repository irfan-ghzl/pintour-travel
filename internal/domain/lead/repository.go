package lead

import (
	"context"
	"errors"
)

// ErrNotConvertible is returned when a lead is not in a state it can be
// converted from — most often because it has been converted already.
//
// Lead.ConvertToParticipant raises it up front, and MarkConverted raises it
// again at the write: a conversion is at most one participant per lead, and the
// second check is what says so when two requests race past the first.
var ErrNotConvertible = errors.New("lead tidak dalam status 'deal'")

type Repository interface {
	Create(ctx context.Context, l *Lead) error
	Update(ctx context.Context, l *Lead) error
	GetByID(ctx context.Context, id string) (*Lead, error)
	List(ctx context.Context, f Filter) ([]Lead, int, error)
	// UpdateStatus moves a lead and records who moved it, in one operation —
	// FR-CRM-02 is not met by a status that changed and a trail that might have
	// been written afterwards.
	UpdateStatus(ctx context.Context, id, status, changedBy string) error
	AssignTo(ctx context.Context, leadID, consultantID string) error
	// MarkConverted also records the transition, with changedBy as its actor.
	MarkConverted(ctx context.Context, leadID, changedBy string) error
	// ListStatusHistory returns the lead's transitions, oldest first.
	ListStatusHistory(ctx context.Context, leadID string) ([]StatusChange, error)
	CountActiveByConsultant(ctx context.Context, consultantID string) (int, error)
}

type NoteRepository interface {
	Create(ctx context.Context, n *Note) error
	ListByLead(ctx context.Context, leadID string) ([]Note, error)
}
