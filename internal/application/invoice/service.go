package invoice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/config"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/uow"
	"github.com/irfan-ghzl/pintour-travel/internal/format"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// ErrNotOwned is returned when an invoice does not belong to the requesting
// participant. Handlers should map this to 404 (not 403) so the existence of
// other participants' invoices is not revealed.
var ErrNotOwned = errors.New("invoice not found")

// ErrInvoiceAlreadyPaid is returned when a fully-paid invoice is sent to the
// gateway. It is the domain's own sentinel: "this invoice is settled" is one
// condition, and a caller matching on it should not have to know whether the
// confirm path or the gateway path raised it.
var ErrInvoiceAlreadyPaid = domainInvoice.ErrAlreadySettled

// ErrProofNotForInvoice is returned when a payment proof is reviewed against an
// invoice it was not uploaded against.
var ErrProofNotForInvoice = errors.New("bukti bayar bukan milik invoice ini")

// Service handles invoice business logic.
type Service struct {
	invoices      domainInvoice.Repository
	proofs        domainInvoice.PaymentProofRepository
	participants  domainParticipant.Repository
	gatewayOrders domainInvoice.GatewayOrderRepository
	// unit runs a gateway settlement as one all-or-nothing operation: claiming
	// the notification and recording the payment it authorises cannot come apart.
	unit     uow.Runner
	fonnte   *service.FonnteService
	pdf      *service.PDFService
	email    *service.EmailService
	midtrans *service.MidtransService
}

func NewService(
	invoices domainInvoice.Repository,
	proofs domainInvoice.PaymentProofRepository,
	participants domainParticipant.Repository,
	gatewayOrders domainInvoice.GatewayOrderRepository,
	unit uow.Runner,
	fonnte *service.FonnteService,
	pdf *service.PDFService,
	email *service.EmailService,
	midtrans *service.MidtransService,
) *Service {
	return &Service{
		invoices: invoices, proofs: proofs, participants: participants,
		gatewayOrders: gatewayOrders, unit: unit,
		fonnte: fonnte, pdf: pdf, email: email, midtrans: midtrans,
	}
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
		// Angka saja — templat WhatsApp yang menuliskan "Rp", sama seperti yang
		// sudah dilakukan jalur pembayaran diterima. Menambahkannya di sini juga
		// membuat pesan yang sampai ke peserta berbunyi "Rp Rp 189.000.000".
		amount := format.Rupiah(inv.Amount)
		dueDate := inv.DueDate.Format("02 Jan 2006")
		// Menunjuk peserta ke portal di sini adalah janji yang belum bisa
		// ditepati: portal baru terbuka setelah pembayaran dikonfirmasi, sehingga
		// invoice ini justru tiba pada satu-satunya masa ketika portalnya
		// tertutup. Peserta yang menurut akan ditolak login dan menyimpulkan
		// passwordnya salah. Nomor, nominal, dan jatuh tempo sudah ada di pesan
		// ini; yang belum ada cukup dikatakan apa adanya.
		pdfLink := inv.PDFPath
		if pdfLink == "" {
			pdfLink = "(tersedia di portal setelah pembayaran dikonfirmasi)"
		}
		_ = s.fonnte.SendInvoice(bgCtx, pt.Phone, pt.Name,
			inv.InvoiceNumber, amount, dueDate, pdfLink, inv.ID)
		// §3.2 email invoice (formal, tabel rincian).
		if s.email != nil && pt.Email != "" {
			_ = s.email.SendEmailInvoice(bgCtx, pt.Email, pt.Name, inv.InvoiceNumber,
				pt.PackageName, format.Rupiah(inv.Amount), dueDate, config.PortalBaseURL()+"/portal")
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

// ConfirmPayment marks an invoice paid in full by hand, for the cases no proof
// covers — cash at the office, a transfer reconciled off-system.
//
// It no longer sends the document request itself. That message belongs to the
// moment an invoice becomes fully paid, and it is sent there, by whichever path
// got it there. Sending it here as well meant an admin who confirmed an invoice
// that a proof had already settled sent the participant the same instructions
// twice.
//
// An invoice already settled is left alone rather than confirmed again, so the
// second confirmation is a no-op instead of a second round of notifications.
func (s *Service) ConfirmPayment(ctx context.Context, invoiceID, confirmedBy, portalBaseURL string) error {
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	// §14.4 Invoice.ConfirmPayment owns the rule; the repository owns the write.
	// An invoice already settled comes back as ErrAlreadySettled, which is a
	// no-op here rather than a failure — the money did arrive, it just must not
	// be announced to the participant a second time.
	if err := inv.ConfirmPayment(confirmedBy, time.Now()); err != nil {
		if errors.Is(err, ErrInvoiceAlreadyPaid) {
			return nil
		}
		return err
	}
	if err := s.invoices.Confirm(ctx, invoiceID, confirmedBy); err != nil {
		return err
	}
	if err := s.participants.Activate(ctx, inv.ParticipantID); err != nil {
		return err
	}
	(&settlement{
		invoice: inv, amount: inv.Amount, fullyPaid: true, portalBaseURL: portalBaseURL,
	}).notify(s)
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
	// The proof has to be one of THIS invoice's. Without the check, a mistyped
	// id settles whichever invoice was named while approving a proof that was
	// never attached to it — money credited to the wrong participant, with both
	// records looking plausible afterwards.
	proof, err := s.proofs.GetByID(ctx, proofID)
	if err != nil {
		return err
	}
	if proof.InvoiceID != invoiceID {
		return ErrProofNotForInvoice
	}

	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	switch status {
	case "ditolak":
		if err := s.proofs.Review(ctx, proofID, status, reviewedBy, notes); err != nil {
			return err
		}
		(&settlement{invoice: inv, rejected: true, reason: notes, portalBaseURL: portalBaseURL}).notify(s)
		return nil
	case "disetujui":
		settled, err := s.settleApprovedProof(ctx, s.ownRepos(), inv, proofID, reviewedBy, notes)
		if err != nil {
			return err
		}
		settled.portalBaseURL = portalBaseURL
		settled.notify(s)
		return nil
	}
	return nil
}

// ownRepos binds the shared settlement logic to the service's own repositories,
// for the paths that are not running inside a unit of work.
func (s *Service) ownRepos() uow.Repos {
	return uow.Repos{
		Participants:  s.participants,
		Invoices:      s.invoices,
		Proofs:        s.proofs,
		GatewayOrders: s.gatewayOrders,
	}
}

// settleApprovedProof approves one proof and brings the invoice up to date: the
// paid amount is recomputed from every approved proof, and the invoice settles
// only once that covers it.
//
// Both the admin's approval and the gateway's automatic one go through here, so
// a partial payment behaves the same whichever way it arrived.
func (s *Service) settleApprovedProof(
	ctx context.Context, repos uow.Repos,
	inv *domainInvoice.Invoice, proofID, reviewedBy, notes string,
) (*settlement, error) {
	if err := repos.Proofs.Review(ctx, proofID, "disetujui", reviewedBy, notes); err != nil {
		return nil, err
	}
	proofs, err := repos.Proofs.GetByInvoice(ctx, inv.ID)
	if err != nil {
		return nil, err
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

	// §14.4 Invoice.IsFullyPaid — the same rule the gateway path applies, so a
	// payment cannot count as full here and partial there.
	fullyPaid := inv.IsFullyPaid(paid)
	switch {
	case fullyPaid && inv.Status != domainInvoice.StatusSettled:
		if err := repos.Invoices.Confirm(ctx, inv.ID, reviewedBy); err != nil {
			return nil, err
		}
		if err := repos.Participants.Activate(ctx, inv.ParticipantID); err != nil {
			return nil, err
		}
	case !fullyPaid && paid > 0 && inv.Status != "dibayar":
		// FR-INV-03's sequence is diterbitkan → menunggu_bayar → dibayar → lunas,
		// and "dibayar" was never reached: an approved-but-partial payment left
		// the invoice on "menunggu_bayar", indistinguishable from one nobody had
		// paid a rupiah towards. It means money accepted, balance outstanding.
		inv.Status = "dibayar"
		if err := repos.Invoices.Update(ctx, inv); err != nil {
			return nil, err
		}
	}
	return &settlement{invoice: inv, amount: thisClaim, fullyPaid: fullyPaid}, nil
}

// settlement is what a completed settlement owes the outside world. It is
// collected inside the unit and delivered after it commits, so nobody is told
// about a payment that was rolled back.
type settlement struct {
	invoice       *domainInvoice.Invoice
	amount        float64
	fullyPaid     bool
	rejected      bool
	reason        string
	portalBaseURL string
}

func (st *settlement) notify(s *Service) {
	if st == nil {
		return
	}
	base := st.portalBaseURL
	if base == "" {
		base = config.PortalBaseURL()
	}
	if st.rejected {
		safe.Go("notifikasi pembayaran ditolak", func() {
			s.notifyPaymentRejected(st.invoice, st.reason, base)
		})
		return
	}
	safe.Go("notifikasi pembayaran diterima", func() {
		s.notifyPaymentReceived(st.invoice, st.amount, st.fullyPaid, base)
	})
}

func (s *Service) notifyPaymentReceived(inv *domainInvoice.Invoice, amountClaimed float64, fullyPaid bool, portalBaseURL string) {
	ctx := context.Background()
	pt, err := s.participants.GetByID(ctx, inv.ParticipantID)
	if err != nil {
		return
	}
	amount := format.Rupiah(amountClaimed)
	_ = s.fonnte.SendPaymentReceived(ctx, pt.Phone, pt.Name, amount, pt.PackageName, inv.ID)
	if s.email != nil && pt.Email != "" {
		_ = s.email.SendEmailPaymentReceived(ctx, pt.Email, pt.Name, inv.InvoiceNumber, amount, time.Now().Format("02 Jan 2006"))
	}
	if fullyPaid {
		portalLink := portalBaseURL + "/portal"
		_ = s.fonnte.SendPortalActivated(ctx, pt.Phone, pt.Name, pt.Phone, portalLink, pt.ID)
		// FR-AUTO-04 and diagram §14.5.3: the participant gets the document list
		// over WhatsApp once payment is confirmed. Only the old manual-confirm
		// endpoint sent it, so a payment settled by an approved proof or by the
		// gateway — which is most of them — told the participant nothing about
		// what to do next. It lives here because this is where "the invoice is
		// now fully paid" is decided, whichever path decided it.
		_ = s.fonnte.SendDocRequest(ctx, pt.Phone, pt.Name, pt.PackageName,
			portalLink+"/documents", pt.ID)
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
	remaining := inv.RemainingBalance(paid) // §14.4 Invoice.RemainingBalance
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
	// Sessions accumulate rather than overwrite, so a payment completed in a tab
	// opened earlier still finds its invoice.
	if err := s.gatewayOrders.Create(ctx, &domainInvoice.GatewayOrder{
		InvoiceID: inv.ID, OrderID: orderID, SnapToken: token,
	}); err != nil {
		return "", "", err
	}
	// The invoice's own columns stay in sync for reads that show "the current
	// session"; the orders table is what a notification is resolved against.
	if err := s.invoices.SetSnap(ctx, inv.ID, token, orderID); err != nil {
		return "", "", err
	}
	return token, s.midtrans.ClientKey(), nil
}

// HandleGatewayNotification processes a verified Midtrans webhook notification.
// Settlement auto-creates an approved payment proof and runs the normal settle
// path (lunas/partial + portal activation + notifications).
// HandleGatewayNotification is idempotent against the identity of the
// transaction it reports, not against the state of the invoice.
//
// It used to return early only when the invoice was already "lunas". A partially
// paid invoice is still "menunggu_bayar", so every redelivery of the same
// settlement created another proof for the same money: four retries of one down
// payment could settle the invoice in full without another rupiah arriving. The
// claim below is what makes a redelivery a no-op, and it lives in the same unit
// as the proof it authorises — a claim whose proof was never written would lose
// a payment permanently, since the gateway would never send it again.
func (s *Service) HandleGatewayNotification(ctx context.Context, n domainInvoice.GatewayNotification, paymentType, grossAmount, fraudStatus string) error {
	invoiceID, err := s.gatewayOrders.FindInvoiceIDByOrder(ctx, n.OrderID)
	if err != nil {
		// "No such order" and "the database is unreachable" are different
		// answers: the first is final, the second must be reported so the
		// gateway retries instead of treating the payment as delivered.
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotOwned
		}
		return fmt.Errorf("resolve gateway order %q: %w", n.OrderID, err)
	}
	n.InvoiceID = invoiceID

	// Fraud review holds a transaction whatever it was paid with. The old guard
	// excluded bank transfers, which let exactly the transactions a reviewer had
	// flagged through untouched.
	held := fraudStatus != "" && fraudStatus != "accept"

	var settled *settlement
	if err := s.unit.Do(ctx, func(ctx context.Context, repos uow.Repos) error {
		claimed, err := repos.GatewayOrders.ClaimNotification(ctx, n)
		if err != nil {
			return err
		}
		if !claimed {
			return errNotificationAlreadyApplied
		}
		settled, err = s.applyGatewayNotification(ctx, repos, n, paymentType, grossAmount, held)
		return err
	}); err != nil {
		if errors.Is(err, errNotificationAlreadyApplied) {
			return nil // a redelivery of something already recorded
		}
		return err
	}

	// Outside the unit: nothing here can be undone, and none of it should hold
	// the gateway's connection open.
	if settled != nil {
		settled.notify(s)
	}
	return nil
}

// errNotificationAlreadyApplied unwinds the unit without treating the redelivery
// as a failure — the claim is rolled back with it, which is correct: the row
// that already held the key is the one that counts.
var errNotificationAlreadyApplied = errors.New("gateway notification already applied")

// applyGatewayNotification records what the notification says, using the
// repositories bound to the unit.
func (s *Service) applyGatewayNotification(
	ctx context.Context, repos uow.Repos,
	n domainInvoice.GatewayNotification, paymentType, grossAmount string, held bool,
) (*settlement, error) {
	inv, err := repos.Invoices.GetByID(ctx, n.InvoiceID)
	if err != nil {
		return nil, err
	}

	switch n.TransactionStatus {
	case "settlement", "capture":
		if held {
			return nil, nil
		}
		amount, _ := strconv.ParseFloat(strings.Split(grossAmount, ".")[0], 64)
		pp := &domainInvoice.PaymentProof{
			InvoiceID:     inv.ID,
			AmountClaimed: amount,
			FilePath:      "midtrans-auto",
			Status:        "disetujui",
			Notes:         "Pembayaran otomatis via Midtrans (" + paymentType + ")",
		}
		if err := repos.Proofs.Create(ctx, pp); err != nil {
			return nil, err
		}
		// Settled as if an admin approved it; inv.IssuedBy is a valid user id for
		// confirmed_by.
		return s.settleApprovedProof(ctx, repos, inv, pp.ID, inv.IssuedBy, "Auto-verified Midtrans")

	case "pending":
		inv.Status = "menunggu_konfirmasi_gateway"
		return nil, repos.Invoices.Update(ctx, inv)

	case "deny", "expire", "cancel":
		inv.Status = "menunggu_bayar"
		if err := repos.Invoices.Update(ctx, inv); err != nil {
			return nil, err
		}
		return &settlement{invoice: inv, rejected: true,
			reason: "Pembayaran gagal/dibatalkan via payment gateway"}, nil
	}
	return nil, nil
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
