package document

import "context"

type Repository interface {
	Create(ctx context.Context, d *Document) error
	GetByID(ctx context.Context, id string) (*Document, error)
	List(ctx context.Context, f Filter) ([]Document, error)
	ListByParticipant(ctx context.Context, participantID string) ([]Document, error)
	Review(ctx context.Context, id, status, reviewedBy, rejectionReason string) error
	Delete(ctx context.Context, id string) error
}

type CountryRequirementRepository interface {
	List(ctx context.Context, countryCode string) ([]CountryRequirement, error)
	Create(ctx context.Context, r *CountryRequirement) error
	Update(ctx context.Context, r *CountryRequirement) error
	Delete(ctx context.Context, id string) error
}
