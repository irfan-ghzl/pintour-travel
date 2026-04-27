package inquirysvc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/inquiry"
)

// InquiryService orchestrates inquiry use cases.
type InquiryService struct {
	repo            inquiry.Repository
	consultantPhone string
}

// NewInquiryService creates a new InquiryService.
func NewInquiryService(repo inquiry.Repository, consultantPhone string) *InquiryService {
	return &InquiryService{repo: repo, consultantPhone: consultantPhone}
}

// SubmitInquiryInput holds raw form data from the HTTP layer.
type SubmitInquiryInput struct {
	FullName      string
	Email         string
	Phone         string
	Destination   string
	TourPackageID string
	NumPeople     int
	Budget        float64
	DurationDays  int
	DepartureDate *string
	Notes         string
}

// SubmitResult is the response after saving an inquiry.
type SubmitResult struct {
	ID        string
	WALink    string
	CreatedAt string
	Message   string
}

// Submit saves the inquiry and returns a WhatsApp consultation link.
func (s *InquiryService) Submit(ctx context.Context, in SubmitInquiryInput) (*SubmitResult, error) {
	// Resolve optional package title for the WA message.
	var packageTitle string
	if in.TourPackageID != "" {
		title, err := s.repo.GetPackageTitle(ctx, in.TourPackageID)
		if err == nil {
			packageTitle = title
		}
	}

	var tourPkgID *string
	if in.TourPackageID != "" {
		tourPkgID = &in.TourPackageID
	}

	departureDate := ""
	if in.DepartureDate != nil {
		departureDate = *in.DepartureDate
	}

	waLink := buildWhatsAppLink(s.consultantPhone, in.FullName, in.Destination,
		packageTitle, in.NumPeople, in.DurationDays, departureDate, in.Budget, in.Notes)

	id, createdAt, err := s.repo.Create(ctx, inquiry.CreateParams{
		FullName:      in.FullName,
		Email:         in.Email,
		Phone:         in.Phone,
		Destination:   in.Destination,
		TourPackageID: tourPkgID,
		NumPeople:     in.NumPeople,
		Budget:        in.Budget,
		DurationDays:  in.DurationDays,
		DepartureDate: in.DepartureDate,
		Notes:         in.Notes,
		WALink:        waLink,
	})
	if err != nil {
		return nil, err
	}

	return &SubmitResult{
		ID:        id,
		WALink:    waLink,
		CreatedAt: createdAt,
		Message:   fmt.Sprintf("Terima kasih %s! Klik link WhatsApp untuk melanjutkan konsultasi.", in.FullName),
	}, nil
}

func (s *InquiryService) ListInquiries(ctx context.Context, f inquiry.Filter) ([]inquiry.Inquiry, int, error) {
	return s.repo.List(ctx, f)
}

// ValidStatuses are allowed inquiry status values.
var ValidStatuses = map[string]bool{
	"new": true, "contacted": true, "in_progress": true,
	"quoted": true, "booked": true, "closed": true,
}

func (s *InquiryService) UpdateStatus(ctx context.Context, id, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

// buildWhatsAppLink generates a pre-filled WhatsApp chat URL.
func buildWhatsAppLink(phone, fullName, destination, packageTitle string,
	numPeople, durationDays int, departureDate string, budget float64, notes string) string {

	lines := []string{
		"Halo, saya ingin konsultasi paket wisata:",
		"",
		fmt.Sprintf("• Nama: %s", fullName),
	}
	if packageTitle != "" {
		lines = append(lines, fmt.Sprintf("• Paket: %s", packageTitle))
	}
	if destination != "" {
		lines = append(lines, fmt.Sprintf("• Destinasi: %s", destination))
	}
	if numPeople > 0 {
		lines = append(lines, fmt.Sprintf("• Jumlah orang: %d", numPeople))
	}
	if durationDays > 0 {
		lines = append(lines, fmt.Sprintf("• Durasi: %d hari", durationDays))
	}
	if departureDate != "" {
		lines = append(lines, fmt.Sprintf("• Tanggal keberangkatan: %s", departureDate))
	}
	if budget > 0 {
		lines = append(lines, fmt.Sprintf("• Budget: Rp %.0f", budget))
	}
	if notes != "" {
		lines = append(lines, fmt.Sprintf("• Catatan: %s", notes))
	}

	return fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(strings.Join(lines, "\n")))
}
