package bookingsvc

import (
	"context"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/booking"
)

// BookingService orchestrates booking use cases.
type BookingService struct {
	repo booking.Repository
}

// NewBookingService creates a new BookingService.
func NewBookingService(repo booking.Repository) *BookingService {
	return &BookingService{repo: repo}
}

func (s *BookingService) ListBookings(ctx context.Context, f booking.Filter) ([]booking.Booking, int, error) {
	return s.repo.List(ctx, f)
}

func (s *BookingService) GetBooking(ctx context.Context, id string) (*booking.Detail, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BookingService) CreateBooking(ctx context.Context, p booking.CreateParams) (id, code string, err error) {
	return s.repo.Create(ctx, p)
}

// ValidPaymentStatuses are the allowed values for payment_status.
var ValidPaymentStatuses = map[string]bool{
	"pending": true, "dp": true, "lunas": true, "refund": true,
}

// ValidBookingStatuses are the allowed values for booking_status.
var ValidBookingStatuses = map[string]bool{
	"confirmed": true, "awaiting_docs": true, "visa_process": true,
	"ready_to_depart": true, "departed": true, "completed": true, "cancelled": true,
}

func (s *BookingService) UpdatePaymentStatus(ctx context.Context, id, status string) error {
	return s.repo.UpdatePaymentStatus(ctx, id, status)
}

func (s *BookingService) UpdateBookingStatus(ctx context.Context, id, status string) error {
	return s.repo.UpdateBookingStatus(ctx, id, status)
}

func (s *BookingService) SetTourLeader(ctx context.Context, id, leaderID string) error {
	return s.repo.SetTourLeader(ctx, id, leaderID)
}

func (s *BookingService) SetWAGroup(ctx context.Context, id, link string) error {
	return s.repo.SetWAGroup(ctx, id, link)
}

func (s *BookingService) SetBriefingDone(ctx context.Context, id string, done bool) error {
	return s.repo.SetBriefingDone(ctx, id, done)
}

func (s *BookingService) DeleteBooking(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
