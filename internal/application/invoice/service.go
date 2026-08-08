package invoice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// portalBaseURL returns the participant portal base URL from env (fallback dev).
func portalBaseURL() string {
	if v := os.Getenv("PORTAL_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:3000"
}

// ErrNotOwned is returned when an invoice does not belong to the requesting
// participant. Handlers should map this to 404 (not 403) so the existence of
// other participants' invoices is not revealed.
var ErrNotOwned = errors.New("invoice not found")

// ErrInvoiceAlreadyPaid is returned when a fully-paid invoice is sent to the gateway.
var ErrInvoiceAlreadyPaid = errors.New("invoice sudah lunas")

// Service handles invoice business logic.
type Service struct {
	invoices     domainInvoice.Repository
	proofs       domainInvoice.PaymentProofRepository
	participants domainParticipant.Repository
	fonnte       *service.FonnteService
	pdf          *service.PDFService
	email        *service.EmailService
	midtrans     *service.MidtransService
}

func NewService(
	invoices domainInvoice.Repository,
	proofs domainInvoice.PaymentProofRepository,
	participants domainParticipant.Repository,
	fonnte *service.FonnteService,
	pdf *service.PDFService,
	email *service.EmailService,
	midtrans *service.MidtransService,
) *Service {
	return &Service{invoices: invoices, proofs: proofs, participants: participants, fonnte: fonnte, pdf: pdf, email: email, midtrans: midtrans}
}

// sumApprovedProofs returns the total amount of approved (disetujui) proofs for
// an invoice — the derived "paid amount" (no paid_amount column, code-wins).
func (s *Service) sumApprovedProofs(ctx context.Context, invoiceID string) (float64, error) {
	proofs, err := s.proofs.GetByInvoice(ctx, invoiceID)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, p := range proofs {
		if p.Status == "disetujui" {
			total += p.AmountClaimed
		}
	}
	return total, nil
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
	safe.Go("kirim invoice ke peserta", func() {
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
		// §3.2 email invoice (formal, tabel rincian).
		if s.email != nil && pt.Email != "" {
			_ = s.email.SendEmailInvoice(bgCtx, pt.Email, pt.Name, inv.InvoiceNumber,
				pt.PackageName, formatRupiah(inv.Amount), dueDate, portalBaseURL()+"/portal")
		}
	})

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

// GetProofWithOwner returns a payment proof together with the participant it
// belongs to. A proof records only the invoice it was uploaded against, so the
// owner has to be resolved through that invoice — which is why an access check
// over a proof cannot be made from the proof alone.
func (s *Service) GetProofWithOwner(ctx context.Context, proofID string) (*domainInvoice.PaymentProof, string, error) {
	proof, err := s.proofs.GetByID(ctx, proofID)
	if err != nil {
		return nil, "", err
	}
	inv, err := s.invoices.GetByID(ctx, proof.InvoiceID)
	if err != nil {
		return nil, "", err
	}
	return proof, inv.ParticipantID, nil
}

// GeneratePDFForParticipant verifies the invoice belongs to participantID
// before generating the PDF. Used by the participant portal to prevent IDOR.
func (s *Service) GeneratePDFForParticipant(ctx context.Context, invoiceID, participantID string) ([]byte, error) {
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil || inv.ParticipantID != participantID {
		return nil, ErrNotOwned
	}
	return s.GeneratePDF(ctx, invoiceID)
}

// UploadProofForParticipant verifies the invoice belongs to participantID
// before attaching a payment proof. Used by the participant portal to prevent IDOR.
func (s *Service) UploadProofForParticipant(ctx context.Context, pp *domainInvoice.PaymentProof, participantID string) error {
	inv, err := s.invoices.GetByID(ctx, pp.InvoiceID)
	if err != nil {
		return err
	}
	if inv == nil || inv.ParticipantID != participantID {
		return ErrNotOwned
	}
	if inv.Status == "diterbitkan" {
		inv.Status = "menunggu_bayar"
		if err := s.invoices.Update(ctx, inv); err != nil {
			return err
		}
	}
	return s.proofs.Create(ctx, pp)
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
	safe.Go("kirim permintaan dokumen", func() {
		bgCtx := context.Background()
		pt, err := s.participants.GetByID(bgCtx, inv.ParticipantID)
		if err != nil {
			return
		}
		portalLink := portalBaseURL + "/portal/documents"
		_ = s.fonnte.SendDocRequest(bgCtx, pt.Phone, pt.Name,
			pt.PackageName, portalLink, pt.ID)
	})
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

// ReviewProofAndSettle records a payment-proof review decision and runs the
// payment automation (prompt §1.4 + §1.3):
//   - rejected → notify participant their proof was rejected.
//   - approved → recompute the paid amount from all approved proofs; when the
//     invoice is fully covered, mark it lunas + activate the portal, otherwise it
//     stays menunggu_bayar (a partial payment). The participant is notified the
//     payment was received either way.
//
// We derive the paid amount from the approved proofs instead of storing a
// paid_amount column, so no schema migration is needed (code-wins decision).
func (s *Service) ReviewProofAndSettle(ctx context.Context, invoiceID, proofID, status, reviewedBy, notes, portalBaseURL string) error {
	if err := s.proofs.Review(ctx, proofID, status, reviewedBy, notes); err != nil {
		return err
	}
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	switch status {
	case "ditolak":
		safe.Go("notifikasi pembayaran ditolak", func() {
			s.notifyPaymentRejected(inv, notes, portalBaseURL)
		})
		return nil
	case "disetujui":
		proofs, err := s.proofs.GetByInvoice(ctx, invoiceID)
		if err != nil {
			return err
		}
		var paid, thisClaim float64
		for _, p := range proofs {
			if p.ID == proofID {
				thisClaim = p.AmountClaimed
			}
			if p.Status == "disetujui" {
				paid += p.AmountClaimed
			}
		}
		fullyPaid := paid >= inv.Amount
		if fullyPaid && inv.Status != "lunas" {
			if err := s.invoices.Confirm(ctx, invoiceID, reviewedBy); err != nil {
				return err
			}
			if err := s.participants.Activate(ctx, inv.ParticipantID); err != nil {
				return err
			}
		}
		safe.Go("notifikasi pembayaran diterima", func() {
			s.notifyPaymentReceived(inv, thisClaim, fullyPaid, portalBaseURL)
		})
		return nil
	}
	return nil
}

func (s *Service) notifyPaymentReceived(inv *domainInvoice.Invoice, amountClaimed float64, fullyPaid bool, portalBaseURL string) {
	ctx := context.Background()
	pt, err := s.participants.GetByID(ctx, inv.ParticipantID)
	if err != nil {
		return
	}
	amount := formatRupiah(amountClaimed)
	_ = s.fonnte.SendPaymentReceived(ctx, pt.Phone, pt.Name, amount, pt.PackageName, inv.ID)
	if s.email != nil && pt.Email != "" {
		_ = s.email.SendEmailPaymentReceived(ctx, pt.Email, pt.Name, inv.InvoiceNumber, amount, time.Now().Format("02 Jan 2006"))
	}
	if fullyPaid {
		portalLink := portalBaseURL + "/portal"
		_ = s.fonnte.SendPortalActivated(ctx, pt.Phone, pt.Name, pt.Phone, portalLink, pt.ID)
		if s.email != nil && pt.Email != "" {
			_ = s.email.SendEmailPortalActivated(ctx, pt.Email, pt.Name, pt.Phone, portalLink)
			// §3.2 doc-request email — daftar dokumen ada di portal (resolved dinamis).
			_ = s.email.SendEmailDocRequest(ctx, pt.Email, pt.Name, pt.PackageName, nil, "H-14 keberangkatan", portalLink+"/documents")
		}
	}
}

// ─── Midtrans payment gateway (v2.0 F1) ──────────────────────────────────────

// CreatePaymentForParticipant creates a Midtrans Snap transaction for the
// remaining balance of an invoice owned by participantID, stores the snap token
// + order id, and returns the token + public client key.
func (s *Service) CreatePaymentForParticipant(ctx context.Context, invoiceID, participantID string) (snapToken, clientKey string, err error) {
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return "", "", err
	}
	if inv == nil || inv.ParticipantID != participantID {
		return "", "", ErrNotOwned
	}
	if s.midtrans == nil || !s.midtrans.Enabled() {
		return "", "", fmt.Errorf("payment gateway belum dikonfigurasi")
	}

	paid, err := s.sumApprovedProofs(ctx, invoiceID)
	if err != nil {
		return "", "", err
	}
	remaining := inv.Amount - paid
	if remaining <= 0 {
		return "", "", ErrInvoiceAlreadyPaid
	}

	pt, err := s.participants.GetByID(ctx, inv.ParticipantID)
	if err != nil {
		return "", "", err
	}

	orderID := fmt.Sprintf("%s-%d", inv.InvoiceNumber, time.Now().Unix())
	token, err := s.midtrans.CreateSnap(ctx, service.SnapRequest{
		OrderID:     orderID,
		GrossAmount: int64(remaining),
		ItemName:    pt.PackageName,
		Customer:    service.SnapCustomer{Name: pt.Name, Email: pt.Email, Phone: pt.Phone},
		ExpiryHours: 24,
	})
	if err != nil {
		return "", "", err
	}
	if err := s.invoices.SetSnap(ctx, inv.ID, token, orderID); err != nil {
		return "", "", err
	}
	return token, s.midtrans.ClientKey(), nil
}

// HandleGatewayNotification processes a verified Midtrans webhook notification.
// Settlement auto-creates an approved payment proof and runs the normal settle
// path (lunas/partial + portal activation + notifications).
func (s *Service) HandleGatewayNotification(ctx context.Context, orderID, txStatus, fraudStatus, paymentType, grossAmount string) error {
	inv, err := s.invoices.GetByOrderID(ctx, orderID)
	if err != nil {
		return ErrNotOwned
	}
	if inv.Status == "lunas" { // idempotent — already settled
		return nil
	}

	switch txStatus {
	case "settlement", "capture":
		if fraudStatus != "" && fraudStatus != "accept" && paymentType != "bank_transfer" {
			return nil // held by fraud review
		}
		amount, _ := strconv.ParseFloat(strings.Split(grossAmount, ".")[0], 64)
		pp := &domainInvoice.PaymentProof{
			InvoiceID:     inv.ID,
			AmountClaimed: amount,
			FilePath:      "midtrans-auto",
			Notes:         "Pembayaran otomatis via Midtrans (" + paymentType + ")",
		}
		if err := s.proofs.Create(ctx, pp); err != nil {
			return err
		}
		// Settle as if an admin approved it; inv.IssuedBy is a valid user for confirmed_by.
		return s.ReviewProofAndSettle(ctx, inv.ID, pp.ID, "disetujui", inv.IssuedBy, "Auto-verified Midtrans", portalBaseURL())

	case "pending":
		inv.Status = "menunggu_konfirmasi_gateway"
		return s.invoices.Update(ctx, inv)

	case "deny", "expire", "cancel":
		inv.Status = "menunggu_bayar"
		if err := s.invoices.Update(ctx, inv); err != nil {
			return err
		}
		safe.Go("notifikasi pembayaran gagal di gateway", func() {
			s.notifyPaymentRejected(inv, "Pembayaran gagal/dibatalkan via payment gateway", portalBaseURL())
		})
		return nil
	}
	return nil
}

func (s *Service) notifyPaymentRejected(inv *domainInvoice.Invoice, reason, portalBaseURL string) {
	ctx := context.Background()
	pt, err := s.participants.GetByID(ctx, inv.ParticipantID)
	if err != nil {
		return
	}
	portalLink := portalBaseURL + "/portal/invoices"
	_ = s.fonnte.SendPaymentRejected(ctx, pt.Phone, pt.Name, reason, portalLink, inv.ID)
	if s.email != nil && pt.Email != "" {
		_ = s.email.SendEmailPaymentRejected(ctx, pt.Email, pt.Name, inv.InvoiceNumber, reason, portalLink)
	}
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
