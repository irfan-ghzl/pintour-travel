package document

import "context"

type Repository interface {
	Create(ctx context.Context, d *Document) error
	GetByID(ctx context.Context, id string) (*Document, error)
	// List returns one page of documents and the total matching the filter.
	List(ctx context.Context, f Filter) ([]Document, int, error)
	ListByParticipant(ctx context.Context, participantID string) ([]Document, error)
	// SummaryByParticipants counts every document each named participant has, by
	// status. The review queue shows "N of M approved" beside each row, and that
	// figure describes the participant — not the page it is being read on, and
	// not the status filter in front of it. One aggregate rather than a query per
	// row, because the queue is paginated and the counts are not.
	SummaryByParticipants(ctx context.Context, participantIDs []string) (map[string]StatusSummary, error)
	Review(ctx context.Context, id, status, reviewedBy, rejectionReason string) error
	Delete(ctx context.Context, id string) error
}

type OCRResultRepository interface {
	Create(ctx context.Context, r *OCRResult) error
	GetByDocument(ctx context.Context, documentID string) (*OCRResult, error)
	// LatestPassportExpiry returns the most recent OCR-extracted passport expiry
	// date (YYYY-MM-DD) for a participant, or "" when none (v2.0 FR-PORTAL-11).
	LatestPassportExpiry(ctx context.Context, participantID string) (string, error)
}

type CountryRequirementRepository interface {
	List(ctx context.Context, countryCode string) ([]CountryRequirement, error)
	Create(ctx context.Context, r *CountryRequirement) error
	Update(ctx context.Context, r *CountryRequirement) error
	Delete(ctx context.Context, id string) error
}
