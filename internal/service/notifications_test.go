package service

// Ticket 13 — the notification templates of §17.1, checked against a test
// server rather than only against the guard that returns early when nothing is
// configured.
//
// These are the messages participants and admins actually receive. Each one is
// asserted on what reaches the recipient: the right address, a subject that
// says what happened, and the details the message exists to carry — an invoice
// number, an amount, a link the reader is meant to follow.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
)

// ─── Email (§17.1, Resend) ────────────────────────────────────────────────────

// sentEmail is one message the fake Resend endpoint received.
type sentEmail struct {
	To      []string `json:"to"`
	From    string   `json:"from"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// emailRecorder stands in for Resend and keeps what was sent.
type emailRecorder struct {
	mu   sync.Mutex
	sent []sentEmail
	srv  *httptest.Server
}

func newEmailRecorder(t *testing.T) (*EmailService, *emailRecorder) {
	t.Helper()
	rec := &emailRecorder{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var msg sentEmail
		_ = json.Unmarshal(body, &msg)
		rec.mu.Lock()
		rec.sent = append(rec.sent, msg)
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(rec.srv.Close)
	return NewEmailService("test-key", "Pintour <noreply@pintour.test>",
		WithEmailBaseURL(rec.srv.URL)), rec
}

// last returns the most recent message, failing the test when none was sent.
func (r *emailRecorder) last(t *testing.T) sentEmail {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sent) == 0 {
		t.Fatal("tidak ada email yang terkirim")
	}
	return r.sent[len(r.sent)-1]
}

// Every §17.1 email reaches the address it was addressed to, carries a subject,
// and contains the details it exists to deliver.
func TestEmailTemplates_CarryTheirDetails(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		send    func(*EmailService) error
		to      string
		subject string
		// mentions are the things the reader needs to find in the body.
		mentions []string
	}{
		"invoice diterbitkan": {
			send: func(s *EmailService) error {
				return s.SendEmailInvoice(ctx, "budi@pintour.test", "Budi", "INV-202608-0001",
					"Umroh Reguler 9 Hari", "25.000.000", "15 Agu 2026", "https://portal.test/portal")
			},
			to:       "budi@pintour.test",
			subject:  "Invoice",
			mentions: []string{"INV-202608-0001", "25.000.000", "Budi"},
		},
		"pembayaran diterima": {
			send: func(s *EmailService) error {
				return s.SendEmailPaymentReceived(ctx, "budi@pintour.test", "Budi",
					"INV-202608-0001", "10.000.000", "09 Agu 2026")
			},
			to:       "budi@pintour.test",
			mentions: []string{"INV-202608-0001", "10.000.000"},
		},
		"pembayaran ditolak": {
			send: func(s *EmailService) error {
				return s.SendEmailPaymentRejected(ctx, "budi@pintour.test", "Budi",
					"INV-202608-0001", "Nominal tidak sesuai", "https://portal.test/portal/invoices")
			},
			to:       "budi@pintour.test",
			mentions: []string{"INV-202608-0001", "Nominal tidak sesuai"},
		},
		"invoice jatuh tempo": {
			send: func(s *EmailService) error {
				return s.SendEmailPaymentOverdue(ctx, "budi@pintour.test", "Budi",
					"INV-202608-0001", "Tim Pintour")
			},
			to:       "budi@pintour.test",
			mentions: []string{"INV-202608-0001", "Tim Pintour"},
		},
		"permintaan dokumen": {
			send: func(s *EmailService) error {
				return s.SendEmailDocRequest(ctx, "budi@pintour.test", "Budi",
					"Umroh Reguler 9 Hari", []string{"Paspor", "KTP"}, "H-14 keberangkatan",
					"https://portal.test/portal/documents")
			},
			to:       "budi@pintour.test",
			mentions: []string{"Paspor", "KTP", "portal/documents"},
		},
		"portal aktif": {
			send: func(s *EmailService) error {
				return s.SendEmailPortalActivated(ctx, "budi@pintour.test", "Budi",
					"628222222222", "https://portal.test/portal")
			},
			to:       "budi@pintour.test",
			mentions: []string{"628222222222", "portal"},
		},
		"briefing dibuka": {
			send: func(s *EmailService) error {
				return s.SendEmailBriefingActivated(ctx, "budi@pintour.test", "Budi",
					"Umroh Reguler 9 Hari", "Ahmad Rizki", "23 Agu 2026",
					"https://portal.test/portal/briefing")
			},
			to:       "budi@pintour.test",
			mentions: []string{"Umroh Reguler 9 Hari", "Ahmad Rizki", "portal/briefing"},
		},
		"pengingat H-14": {
			send: func(s *EmailService) error {
				return s.SendEmailReminderH14(ctx, "budi@pintour.test", "Budi",
					"Umroh Reguler 9 Hari", "23 Agu 2026", "https://portal.test/portal")
			},
			to:       "budi@pintour.test",
			mentions: []string{"23 Agu 2026"},
		},
		"lead baru ke admin": {
			send: func(s *EmailService) error {
				return s.SendEmailAdminNewLeads(ctx, "admin@pintour.test", "Prospek Satu",
					"628111111111", "prospek@pintour.test", "Umroh Reguler 9 Hari",
					"https://app.test/admin/leads")
			},
			to: "admin@pintour.test",
			// A consultant needs the package before picking up the phone.
			mentions: []string{"Prospek Satu", "628111111111", "Umroh Reguler 9 Hari"},
		},
		"peringatan kuota ke admin": {
			send: func(s *EmailService) error {
				return s.SendEmailAdminQuotaWarning(ctx, "admin@pintour.test",
					"Umroh Reguler 9 Hari", "23 Agu 2026", 3, 40)
			},
			to:       "admin@pintour.test",
			mentions: []string{"Umroh Reguler 9 Hari", "3", "40"},
		},
		"reset password": {
			send: func(s *EmailService) error {
				return s.SendResetPassword(ctx, "admin@pintour.test", "Admin",
					"https://app.test/reset?token=abc")
			},
			to:       "admin@pintour.test",
			mentions: []string{"token=abc"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc, rec := newEmailRecorder(t)
			if err := tc.send(svc); err != nil {
				t.Fatalf("kirim: %v", err)
			}
			msg := rec.last(t)
			if len(msg.To) != 1 || msg.To[0] != tc.to {
				t.Errorf("penerima = %v, want [%s]", msg.To, tc.to)
			}
			if msg.Subject == "" {
				t.Error("email terkirim tanpa subjek")
			}
			if tc.subject != "" && !strings.Contains(msg.Subject, tc.subject) {
				t.Errorf("subjek %q tidak menyebut %q", msg.Subject, tc.subject)
			}
			if msg.From == "" {
				t.Error("email terkirim tanpa alamat pengirim")
			}
			for _, want := range tc.mentions {
				if !strings.Contains(msg.HTML, want) {
					t.Errorf("isi email tidak memuat %q", want)
				}
			}
		})
	}
}

// The overdue digest lists every invoice it was given, so an admin sees the
// whole backlog in one message instead of one email per participant.
func TestEmailAdminOverdueDigest_ListsEveryInvoice(t *testing.T) {
	svc, rec := newEmailRecorder(t)

	rows := []OverdueInvoiceRow{
		{ParticipantName: "Budi", InvoiceNumber: "INV-202608-0001", Amount: "25.000.000"},
		{ParticipantName: "Siti", InvoiceNumber: "INV-202608-0002", Amount: "12.500.000"},
	}
	if err := svc.SendEmailAdminPaymentOverdue(context.Background(),
		"admin@pintour.test", rows, "https://app.test/admin/invoices"); err != nil {
		t.Fatalf("kirim: %v", err)
	}

	msg := rec.last(t)
	for _, want := range []string{"Budi", "Siti", "INV-202608-0001", "INV-202608-0002", "25.000.000"} {
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("digest tidak memuat %q", want)
		}
	}
}

// Without an API key nothing is sent and nothing fails loudly — a deployment
// with no email provider must not break the flows that notify.
func TestEmail_UnconfiguredSendsNothing(t *testing.T) {
	svc := NewEmailService("", "noreply@pintour.test")
	if err := svc.Send(context.Background(), "budi@pintour.test", "Halo", "<p>Halo</p>"); err == nil {
		t.Error("Send tanpa API key = nil, want error yang menyebutkan konfigurasinya")
	}
}

// An upstream refusal is reported rather than swallowed, so a caller that cares
// can log it.
func TestEmail_UpstreamFailureIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	svc := NewEmailService("bad-key", "noreply@pintour.test", WithEmailBaseURL(srv.URL))
	if err := svc.Send(context.Background(), "budi@pintour.test", "Halo", "<p>Halo</p>"); err == nil {
		t.Error("status 401 dari penyedia email tidak dilaporkan")
	}
}

// ─── WhatsApp (§17.1, Fonnte) ─────────────────────────────────────────────────

// waRecorder stands in for the Fonnte gateway.
type waRecorder struct {
	mu   sync.Mutex
	sent []map[string]any
	srv  *httptest.Server
}

func newWARecorder(t *testing.T) (*FonnteService, *waRecorder, *fakeNotificationRepo) {
	t.Helper()
	rec := &waRecorder{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		rec.mu.Lock()
		rec.sent = append(rec.sent, payload)
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":true}`))
	}))
	t.Cleanup(rec.srv.Close)
	repo := &fakeNotificationRepo{}
	return NewFonnteService("test-token", repo, WithFonnteBaseURL(rec.srv.URL)), rec, repo
}

func (r *waRecorder) last(t *testing.T) map[string]any {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sent) == 0 {
		t.Fatal("tidak ada pesan WhatsApp yang terkirim")
	}
	return r.sent[len(r.sent)-1]
}

// Every §17.1 WhatsApp message reaches the number it was addressed to and
// carries the details the recipient needs to act on it.
func TestWhatsAppTemplates_CarryTheirDetails(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		send     func(*FonnteService) error
		phone    string
		mentions []string
	}{
		"selamat datang lead": {
			send: func(s *FonnteService) error {
				return s.SendLeadsWelcome(ctx, "628111111111", "Budi", "Umroh Reguler 9 Hari", "lead-1")
			},
			phone:    "628111111111",
			mentions: []string{"Budi", "Umroh Reguler 9 Hari"},
		},
		"pembayaran ditolak": {
			send: func(s *FonnteService) error {
				return s.SendPaymentRejected(ctx, "628222222222", "Budi",
					"Nominal tidak sesuai", "https://portal.test/portal/invoices", "inv-1")
			},
			phone:    "628222222222",
			mentions: []string{"Nominal tidak sesuai", "portal/invoices"},
		},
		"invoice jatuh tempo": {
			send: func(s *FonnteService) error {
				return s.SendPaymentOverdue(ctx, "628222222222", "Budi",
					"INV-202608-0001", "Tim Pintour", "inv-1")
			},
			phone: "628222222222",
			// FR-AUTO: the reminder names the invoice, so the participant knows
			// which bill is being chased.
			mentions: []string{"INV-202608-0001", "Tim Pintour"},
		},
		"briefing dibuka": {
			send: func(s *FonnteService) error {
				return s.SendBriefingActivated(ctx, "628222222222", "Budi", "Ahmad Rizki",
					"23 Agu 2026", "https://portal.test/portal/briefing", "participant-1")
			},
			phone:    "628222222222",
			mentions: []string{"Ahmad Rizki", "23 Agu 2026", "portal/briefing"},
		},
		"lead menganggur ke konsultan": {
			send: func(s *FonnteService) error {
				return s.SendLeadsStale(ctx, "628333333333", "Konsultan", "Prospek Satu", "lead-1")
			},
			phone:    "628333333333",
			mentions: []string{"Prospek Satu"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc, rec, repo := newWARecorder(t)
			if err := tc.send(svc); err != nil {
				t.Fatalf("kirim: %v", err)
			}
			payload := rec.last(t)
			if got, _ := payload["target"].(string); got != tc.phone {
				t.Errorf("tujuan = %q, want %q", got, tc.phone)
			}
			message, _ := payload["message"].(string)
			for _, want := range tc.mentions {
				if !strings.Contains(message, want) {
					t.Errorf("pesan tidak memuat %q:\n%s", want, message)
				}
			}
			// Every send is recorded, which is what the dedup and the admin's
			// message history both read.
			if repo.count() != 1 {
				t.Errorf("catatan notifikasi = %d, want 1", repo.count())
			}
		})
	}
}

// A gateway that refuses is retried, and the attempt is still recorded — a
// message that never arrived has to be visible in the log, not absent from it.
func TestWhatsApp_FailureIsRecorded(t *testing.T) {
	original := fonnteRetryDelay
	SetRetryDelay(time.Millisecond)
	defer SetRetryDelay(original)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	repo := &fakeNotificationRepo{}
	svc := NewFonnteService("test-token", repo, WithFonnteBaseURL(srv.URL))
	err := svc.SendLeadsWelcome(context.Background(), "628111111111", "Budi", "Umroh", "lead-1")
	if err == nil {
		t.Error("kegagalan gateway tidak dilaporkan")
	}
	if attempts < 2 {
		t.Errorf("percobaan = %d, want lebih dari satu (ada retry)", attempts)
	}
	if repo.count() == 0 {
		t.Error("kegagalan tidak tercatat di riwayat notifikasi")
	}
}

// ─── Fake notification repository ─────────────────────────────────────────────

// fakeNotificationRepo records the WhatsApp log rows the service writes, which
// are what the dedup checks and the admin's message history read.
type fakeNotificationRepo struct {
	mu   sync.Mutex
	rows []notification.WANotification
}

func (r *fakeNotificationRepo) Create(_ context.Context, n *notification.WANotification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n.ID = "notif-" + n.MessageType
	r.rows = append(r.rows, *n)
	return nil
}

func (r *fakeNotificationRepo) UpdateStatus(_ context.Context, id, status string, _ interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.rows {
		if r.rows[i].ID == id {
			r.rows[i].Status = status
		}
	}
	return nil
}

func (r *fakeNotificationRepo) ListByReference(_ context.Context, refID, refType string) ([]notification.WANotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []notification.WANotification
	for _, n := range r.rows {
		if n.ReferenceID != nil && *n.ReferenceID == refID && n.ReferenceType != nil && *n.ReferenceType == refType {
			out = append(out, n)
		}
	}
	return out, nil
}

func (r *fakeNotificationRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.rows)
}
