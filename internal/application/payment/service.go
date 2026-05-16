package paymentsvc

import (
	"context"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/payment"
)

// ValidPaymentTypes are the allowed payment_type values.
var ValidPaymentTypes = map[string]bool{
	"dp": true, "pelunasan": true, "full": true,
}

// PaymentService orchestrates payment use cases.
type PaymentService struct {
	repo payment.Repository
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(repo payment.Repository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ListByBooking(ctx context.Context, bookingID string) ([]payment.Payment, error) {
	return s.repo.ListByBooking(ctx, bookingID)
}

func (s *PaymentService) CreatePayment(ctx context.Context, p payment.CreateParams) (string, error) {
	return s.repo.Create(ctx, p)
}

func (s *PaymentService) VerifyPayment(ctx context.Context, id, verifiedByUserID string) error {
	return s.repo.Verify(ctx, id, verifiedByUserID)
}

func (s *PaymentService) DeletePayment(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
