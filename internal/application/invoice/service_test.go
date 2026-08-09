package invoice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// ─── Fakes ────────────────────────────────────────────────────────────────────

type fakeInvoiceRepo struct {
	createdInvoice *domainInvoice.Invoice
	nextSeq        int
	stored         map[string]*domainInvoice.Invoice
}

func newFakeInvoiceRepo() *fakeInvoiceRepo {
	return &fakeInvoiceRepo{stored: map[string]*domainInvoice.Invoice{}}
}

func (r *fakeInvoiceRepo) Create(_ context.Context, inv *domainInvoice.Invoice) error {
	inv.ID = "inv-" + inv.InvoiceNumber
	inv.CreatedAt = time.Now()
	r.createdInvoice = inv
	r.stored[inv.ID] = inv
	return nil
}
func (r *fakeInvoiceRepo) Update(_ context.Context, inv *domainInvoice.Invoice) error {
	r.stored[inv.ID] = inv
	return nil
}
func (r *fakeInvoiceRepo) GetByID(_ context.Context, id string) (*domainInvoice.Invoice, error) {
	inv, ok := r.stored[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return inv, nil
}
func (r *fakeInvoiceRepo) List(_ context.Context, _ domainInvoice.Filter) ([]domainInvoice.Invoice, int, error) {
	out := make([]domainInvoice.Invoice, 0, len(r.stored))
	for _, inv := range r.stored {
		out = append(out, *inv)
	}
	return out, len(out), nil
}
func (r *fakeInvoiceRepo) NextSequence(_ context.Context, _ string) (int, error) {
	r.nextSeq++
	return r.nextSeq, nil
}
func (r *fakeInvoiceRepo) Confirm(_ context.Context, id, userID string) error {
	if inv, ok := r.stored[id]; ok {
		inv.Status = "lunas"
		now := time.Now()
		inv.ConfirmedAt = &now
		inv.ConfirmedBy = &userID
	}
	return nil
}
func (r *fakeInvoiceRepo) SetSnap(_ context.Context, id, token, orderID string) error {
	if inv, ok := r.stored[id]; ok {
		inv.SnapToken = token
		inv.MidtransOrderID = orderID
	}
	return nil
}

type fakeProofRepo struct{ created int }

func (r *fakeProofRepo) Create(_ context.Context, _ *domainInvoice.PaymentProof) error {
	r.created++
	return nil
}
func (r *fakeProofRepo) GetByID(_ context.Context, _ string) (*domainInvoice.PaymentProof, error) {
	return nil, nil
}
func (r *fakeProofRepo) GetByInvoice(_ context.Context, _ string) ([]domainInvoice.PaymentProof, error) {
	return nil, nil
}
func (r *fakeProofRepo) Review(_ context.Context, _, _, _, _ string) error { return nil }

type fakeParticipantRepo struct {
	activated string
}

func (r *fakeParticipantRepo) Create(_ context.Context, _ *domainParticipant.Participant) error {
	return nil
}
func (r *fakeParticipantRepo) Update(_ context.Context, _ *domainParticipant.Participant) error {
	return nil
}
func (r *fakeParticipantRepo) GetByID(_ context.Context, id string) (*domainParticipant.Participant, error) {
	return &domainParticipant.Participant{
		ID: id, Name: "Test Participant", Phone: "628123456789",
		PackageName: "Test Package", RoomType: "double",
	}, nil
}
func (r *fakeParticipantRepo) GetByPhone(_ context.Context, _ string) (*domainParticipant.Participant, error) {
	return nil, nil
}
func (r *fakeParticipantRepo) List(_ context.Context, _ domainParticipant.Filter) ([]domainParticipant.Participant, int, error) {
	return nil, 0, nil
}
func (r *fakeParticipantRepo) Activate(_ context.Context, id string) error {
	r.activated = id
	return nil
}
func (r *fakeParticipantRepo) SetNIK(_ context.Context, _, _ string) error { return nil }
func (r *fakeParticipantRepo) ListByPortalUser(_ context.Context, _, _ string) ([]domainParticipant.Participant, error) {
	return nil, nil
}
func (r *fakeParticipantRepo) ListByBatch(_ context.Context, _ string) ([]domainParticipant.Participant, error) {
	return nil, nil
}
func (r *fakeParticipantRepo) ListByDepartureDaysAhead(_ context.Context, _ int) ([]domainParticipant.Participant, error) {
	return nil, nil
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCreate_GeneratesInvoiceNumber(t *testing.T) {
	invRepo := newFakeInvoiceRepo()
	proofs := &fakeProofRepo{}
	parts := &fakeParticipantRepo{}
	fonnte := service.NewFonnteService("", nil) // no-op when token empty
	pdf := service.NewPDFService()
	svc := NewService(invRepo, proofs, parts, nil, nil, fonnte, pdf, nil, nil)

	inv := &domainInvoice.Invoice{
		ParticipantID: "p1",
		BatchID:       "b1",
		Amount:        25000000,
		DueDate:       calendar.Today().AddDays(30),
		IssuedBy:      "u1",
	}
	if err := svc.Create(context.Background(), inv); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Format per FR-INV-02: INV-YYYYMM-XXXX
	yearMonth := time.Now().Format("200601")
	expected := fmt.Sprintf("INV-%s-0001", yearMonth)
	if inv.InvoiceNumber != expected {
		t.Errorf("invoice_number = %q, want %q", inv.InvoiceNumber, expected)
	}
	if inv.Status != "diterbitkan" {
		t.Errorf("expected status diterbitkan, got %q", inv.Status)
	}
}

func TestCreate_IncrementsSequence(t *testing.T) {
	invRepo := newFakeInvoiceRepo()
	svc := NewService(invRepo, &fakeProofRepo{}, &fakeParticipantRepo{}, nil, nil,
		service.NewFonnteService("", nil), service.NewPDFService(), nil, nil)

	yearMonth := time.Now().Format("200601")
	expected := []string{
		fmt.Sprintf("INV-%s-0001", yearMonth),
		fmt.Sprintf("INV-%s-0002", yearMonth),
		fmt.Sprintf("INV-%s-0003", yearMonth),
	}

	for i, want := range expected {
		inv := &domainInvoice.Invoice{
			ParticipantID: "p1", BatchID: "b1", Amount: 1000000,
			DueDate: calendar.Today(), IssuedBy: "u1",
		}
		if err := svc.Create(context.Background(), inv); err != nil {
			t.Fatalf("Create #%d error: %v", i+1, err)
		}
		if inv.InvoiceNumber != want {
			t.Errorf("invoice #%d: got %q, want %q", i+1, inv.InvoiceNumber, want)
		}
	}
}

func TestConfirmPayment_ActivatesPortal(t *testing.T) {
	invRepo := newFakeInvoiceRepo()
	parts := &fakeParticipantRepo{}
	svc := NewService(invRepo, &fakeProofRepo{}, parts, nil, nil,
		service.NewFonnteService("", nil), service.NewPDFService(), nil, nil)

	// Create first
	inv := &domainInvoice.Invoice{
		ParticipantID: "participant-123", BatchID: "b1",
		Amount: 5000000, DueDate: calendar.Today(), IssuedBy: "u1",
	}
	if err := svc.Create(context.Background(), inv); err != nil {
		t.Fatal(err)
	}

	if err := svc.ConfirmPayment(context.Background(), inv.ID, "admin1", "http://test"); err != nil {
		t.Fatalf("ConfirmPayment error: %v", err)
	}
	if parts.activated != "participant-123" {
		t.Errorf("expected participant activated, got %q", parts.activated)
	}
}

func TestUploadProof_TransitionsStatus(t *testing.T) {
	invRepo := newFakeInvoiceRepo()
	proofs := &fakeProofRepo{}
	svc := NewService(invRepo, proofs, &fakeParticipantRepo{}, nil, nil,
		service.NewFonnteService("", nil), service.NewPDFService(), nil, nil)

	inv := &domainInvoice.Invoice{
		ParticipantID: "p1", BatchID: "b1",
		Amount: 1000000, DueDate: calendar.Today(), IssuedBy: "u1",
	}
	_ = svc.Create(context.Background(), inv)

	pp := &domainInvoice.PaymentProof{
		InvoiceID: inv.ID, FilePath: "/path", AmountClaimed: 1000000,
	}
	if err := svc.UploadProof(context.Background(), pp); err != nil {
		t.Fatalf("UploadProof error: %v", err)
	}
	stored := invRepo.stored[inv.ID]
	if stored.Status != "menunggu_bayar" {
		t.Errorf("expected status menunggu_bayar after proof upload, got %q", stored.Status)
	}
	if proofs.created != 1 {
		t.Errorf("expected 1 proof created, got %d", proofs.created)
	}
}

// The rupiah formatter moved to internal/format, where it is tested once for
// every caller instead of once per package that happened to copy it.
