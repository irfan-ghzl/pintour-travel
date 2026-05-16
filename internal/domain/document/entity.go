package document

import "context"

// Document represents a file uploaded for a booking participant.
type Document struct {
	ID            string  `json:"id"`
	ParticipantID string  `json:"participant_id"`
	DocType       string  `json:"doc_type"` // passport, ktp, bank_statement, visa_support
	FileURL       string  `json:"file_url"`
	Notes         *string `json:"notes,omitempty"`
	Verified      bool    `json:"verified"`
	VerifiedBy    *string `json:"verified_by,omitempty"`
	VerifiedAt    *string `json:"verified_at,omitempty"`
	UploadedAt    string  `json:"uploaded_at"`
}

// CreateParams contains the data needed to upload a participant document.
type CreateParams struct {
	ParticipantID string
	DocType       string
	FileURL       string
	Notes         *string
}

// Repository is the persistence abstraction for the document domain.
type Repository interface {
	ListByParticipant(ctx context.Context, participantID string) ([]Document, error)
	ListByBooking(ctx context.Context, bookingID string) ([]Document, error)
	Create(ctx context.Context, p CreateParams) (string, error)
	Verify(ctx context.Context, id, verifiedByUserID string) error
	Delete(ctx context.Context, id string) error
}
