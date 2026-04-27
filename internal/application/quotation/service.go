package quotationsvc

import (
	"context"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/quotation"
)

// QuotationService orchestrates quotation use cases.
type QuotationService struct {
	repo quotation.Repository
}

// NewQuotationService creates a new QuotationService.
func NewQuotationService(repo quotation.Repository) *QuotationService {
	return &QuotationService{repo: repo}
}

func (s *QuotationService) Create(ctx context.Context, p quotation.CreateParams) (string, error) {
	return s.repo.Create(ctx, p)
}

func (s *QuotationService) GetByID(ctx context.Context, id string) (*quotation.Detail, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *QuotationService) List(ctx context.Context, f quotation.Filter) ([]quotation.Quotation, int, error) {
	return s.repo.List(ctx, f)
}
