package documentsvc

import (
	"context"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
)

// ValidDocTypes are the allowed document type values.
var ValidDocTypes = map[string]bool{
	"passport": true, "ktp": true, "bank_statement": true, "visa_support": true, "photo": true, "other": true,
}

// DocumentService orchestrates participant document use cases.
type DocumentService struct {
	repo document.Repository
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(repo document.Repository) *DocumentService {
	return &DocumentService{repo: repo}
}

func (s *DocumentService) ListByParticipant(ctx context.Context, participantID string) ([]document.Document, error) {
	return s.repo.ListByParticipant(ctx, participantID)
}

func (s *DocumentService) ListByBooking(ctx context.Context, bookingID string) ([]document.Document, error) {
	return s.repo.ListByBooking(ctx, bookingID)
}

func (s *DocumentService) CreateDocument(ctx context.Context, p document.CreateParams) (string, error) {
	return s.repo.Create(ctx, p)
}

func (s *DocumentService) VerifyDocument(ctx context.Context, id, verifiedByUserID string) error {
	return s.repo.Verify(ctx, id, verifiedByUserID)
}

func (s *DocumentService) DeleteDocument(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
