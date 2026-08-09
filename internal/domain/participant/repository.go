package participant

import "context"

type Repository interface {
	Create(ctx context.Context, p *Participant) error
	Update(ctx context.Context, p *Participant) error
	GetByID(ctx context.Context, id string) (*Participant, error)
	GetByPhone(ctx context.Context, phone string) (*Participant, error)
	// v2.0 F1/F2 — every tour for a portal identity (newest departure first),
	// keyed on portal_user_id with phone as a fallback for unlinked legacy rows.
	ListByPortalUser(ctx context.Context, portalUserID, phone string) ([]Participant, error)
	List(ctx context.Context, f Filter) ([]Participant, int, error)
	Activate(ctx context.Context, id string) error
	SetNIK(ctx context.Context, id, nik string) error // TODO(ocr-v2.0-F3): OCR auto-fill
	ListByBatch(ctx context.Context, batchID string) ([]Participant, error)
	// For scheduler: list participants departing in N days
	ListByDepartureDaysAhead(ctx context.Context, days int) ([]Participant, error)
}
