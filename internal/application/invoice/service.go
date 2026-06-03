package invoice

import (
	"context"
	"fmt"
	"time"

	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// Service handles invoice business logic.
type Service struct {
	invoices     domainInvoice.Repository
	proofs       domainInvoice.PaymentProofRepository
	participants domainParticipant.Repository
	fonnte       *service.FonnteService
	pdf          *service.PDFService
}

func NewService(
	invoices domainInvoice.Repository,
	proofs domainInvoice.PaymentProofRepository,
	participants domainParticipant.Repository,
	fonnte *service.FonnteService,
	pdf *service.PDFService,
) *Service {
	return &Service{invoices: invoices, proofs: proofs, participants: participants, fonnte: fonnte, pdf: pdf}
}

// Create generates a new invoice, generates PDF, and sends WA.
func (s *Service) Create(ctx context.Context, inv *domainInvoice.Invoice) error {
	yearMonth := time.Now().Format("200601")
	seq, err := s.invoices.NextSequence(ctx, yearMonth)
	if err != nil {
		return fmt.Errorf("generate sequence: %w", err)
	}
	inv.InvoiceNumber = fmt.Sprintf("INV-%s-%04d", yearMonth, seq)
	inv.Status = "diterbitkan"

	if err := s.invoices.Create(ctx, inv); err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}

	// Async: generate PDF then send WA
	go func() {
		bgCtx := context.Background()
		pt, err := s.participants.GetByID(bgCtx, inv.ParticipantID)
		if err != nil {
			return
		}
		amount := fmt.Sprintf("Rp %s", formatRupiah(inv.Amount))
		dueDate := inv.DueDate.Format("02 Jan 2006")
		pdfLink := inv.PDFPath
		if pdfLink == "" {
			pdfLink = "(PDF tersedia di portal peserta)"
		}
		_ = s.fonnte.SendInvoice(bgCtx, pt.Phone, pt.Name,
			inv.InvoiceNumber, amount, dueDate, pdfLink, inv.ID)
	}()

	return nil
}

func (s *Service) GetInvoice(ctx context.Context, id string) (*domainInvoice.Invoice, error) {
	return s.invoices.GetByID(ctx, id)
}

func (s *Service) ListInvoices(ctx context.Context, f domainInvoice.Filter) ([]domainInvoice.Invoice, int, error) {
	return s.invoices.List(ctx, f)
}

// GetInvoicesByParticipant lists invoices for a specific participant (used by portal).
func (s *Service) GetInvoicesByParticipant(ctx context.Context, participantID string) ([]domainInvoice.Invoice, error) {
	f := domainInvoice.Filter{ParticipantID: &participantID, Page: 1, PerPage: 50}
	invoices, _, err := s.invoices.List(ctx, f)
	return invoices, err
}

// GeneratePDF generates invoice PDF bytes.
// The path is saved to invoices.pdf_path per §12 on first generation.
func (s *Service) GeneratePDF(ctx context.Context, invoiceID string) ([]byte, error) {
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	pt, err := s.participants.GetByID(ctx, inv.ParticipantID)
	if err != nil {
		return nil, err
	}

	batchDate := "—"
	if pt.BatchDepartureDate != nil {
		batchDate = pt.BatchDepartureDate.Format("02 January 2006")
	}

	data := service.InvoiceData{
		InvoiceNumber:    inv.InvoiceNumber,
		IssuedAt:         inv.CreatedAt,
		DueDate:          inv.DueDate,
		ParticipantName:  inv.ParticipantName,
		ParticipantPhone: inv.ParticipantPhone,
		PackageName:      inv.PackageName,
		BatchDate:        batchDate,
		RoomType:         pt.RoomType,
		Amount:           inv.Amount,
		Notes:            inv.Notes,
		IssuedByName:     inv.IssuedByName,
	}
	pdfBytes, err := s.pdf.GenerateInvoice(data)
	if err != nil {
		return nil, err
	}

	// Save PDF path reference to invoices.pdf_path (§12) if not already set.
	// In production this would be a Supabase Storage URL after upload.
	// For now, store the endpoint path so frontend can reference it.
	if inv.PDFPath == "" {
		inv.PDFPath = fmt.Sprintf("/api/v1/admin/invoices/%s/pdf", inv.ID)
		_ = s.invoices.Update(ctx, inv)
	}

	return pdfBytes, nil
}

// ConfirmPayment confirms payment, activates portal, sends doc request WA.
func (s *Service) ConfirmPayment(ctx context.Context, invoiceID, confirmedBy, portalBaseURL string) error {
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if err := s.invoices.Confirm(ctx, invoiceID, confirmedBy); err != nil {
		return err
	}
	if err := s.participants.Activate(ctx, inv.ParticipantID); err != nil {
		return err
	}
	go func() {
		bgCtx := context.Background()
		pt, err := s.participants.GetByID(bgCtx, inv.ParticipantID)
		if err != nil {
			return
		}
		portalLink := portalBaseURL + "/portal/documents"
		_ = s.fonnte.SendDocRequest(bgCtx, pt.Phone, pt.Name,
			pt.PackageName, portalLink, pt.ID)
	}()
	return nil
}

// UploadProof saves a payment proof and updates invoice status.
func (s *Service) UploadProof(ctx context.Context, pp *domainInvoice.PaymentProof) error {
	inv, err := s.invoices.GetByID(ctx, pp.InvoiceID)
	if err != nil {
		return err
	}
	if inv.Status == "diterbitkan" {
		inv.Status = "menunggu_bayar"
		if err := s.invoices.Update(ctx, inv); err != nil {
			return err
		}
	}
	return s.proofs.Create(ctx, pp)
}

func (s *Service) ListProofs(ctx context.Context, invoiceID string) ([]domainInvoice.PaymentProof, error) {
	return s.proofs.GetByInvoice(ctx, invoiceID)
}

func (s *Service) ReviewProof(ctx context.Context, proofID, status, reviewedBy, notes string) error {
	return s.proofs.Review(ctx, proofID, status, reviewedBy, notes)
}

func formatRupiah(amount float64) string {
	s := fmt.Sprintf("%.0f", amount)
	n := len(s)
	result := make([]byte, 0, n+n/3)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
