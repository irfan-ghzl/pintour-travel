package airport

import "context"

type Repository interface {
	// Upsert creates or returns existing checklist for a participant in a batch.
	InitForBatch(ctx context.Context, batchID string) error
	GetByParticipant(ctx context.Context, participantID, batchID string) (*Checklist, error)
	ListByBatch(ctx context.Context, f Filter) ([]Checklist, error)
	UpdateBaggage(ctx context.Context, participantID, batchID, handledBy string) error
	UpdateTicket(ctx context.Context, participantID, batchID, handledBy string) error
	UpdatePassport(ctx context.Context, participantID, batchID, handledBy string) error
	GetBatchProgress(ctx context.Context, batchID string) (*BatchProgress, error)
}
