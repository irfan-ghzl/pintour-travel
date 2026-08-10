package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
)

// defaultFonnteBaseURL is the gateway every deployment talks to.
const defaultFonnteBaseURL = "https://api.fonnte.com"

// FonnteService sends WhatsApp messages via Fonnte API.
type FonnteService struct {
	apiToken string
	baseURL  string
	repo     notification.Repository
	client   *http.Client
}

// FonnteOption customises the client at construction.
type FonnteOption func(*FonnteService)

// WithFonnteBaseURL points the client somewhere other than the real gateway.
//
// It exists for tests: Send returns before writing its notification row when it
// has no token, so with the endpoint hardcoded there was no way to observe any
// message the system sends — and handing the tests a real token would have put
// them on the network. Given a fake token and a local server, Send now runs the
// whole way through: it writes the row, posts, and updates the status. An empty
// value keeps the default, so no deployment behaviour changes.
func WithFonnteBaseURL(baseURL string) FonnteOption {
	return func(s *FonnteService) {
		if baseURL != "" {
			s.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func NewFonnteService(apiToken string, repo notification.Repository, opts ...FonnteOption) *FonnteService {
	s := &FonnteService{
		apiToken: apiToken,
		baseURL:  defaultFonnteBaseURL,
		repo:     repo,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Enabled reports whether a gateway is configured. Send is a no-op without one,
// so a caller that tells a user "we messaged them" has to ask first — mirrors
// StorageService.Enabled.
func (s *FonnteService) Enabled() bool { return s != nil && s.repo != nil && s.apiToken != "" }

type fonntePayload struct {
	Target      string `json:"target"`
	Message     string `json:"message"`
	CountryCode string `json:"countryCode"`
}

type fonnteResponse struct {
	Status  bool     `json:"status"`
	Reason  string   `json:"reason"`
	ID      []string `json:"id"`
	Process string   `json:"process"`
	Target  []string `json:"target"`
}

// PRD §16.1: Timeout retry maksimal 3 kali dengan jeda 30 detik.
// Use a short interval in tests via SetRetryDelay.
var fonnteRetryDelay = 30 * time.Second

// SetRetryDelay overrides the delay between retries (for tests).
func SetRetryDelay(d time.Duration) { fonnteRetryDelay = d }

// Send dispatches a single WA message and logs it.
// Implements retry (max 3 attempts, 30s delay) on timeout/transient errors per PRD §16.1.
// Returns nil if Fonnte is not configured or repo is missing (development mode).
func (s *FonnteService) Send(ctx context.Context, phone, name, msgType, message string, refID, refType *string) error {
	if s == nil || s.repo == nil || s.apiToken == "" {
		return nil
	}
	n := &notification.WANotification{
		RecipientPhone: phone,
		RecipientName:  name,
		MessageType:    msgType,
		MessageContent: message,
		ReferenceID:    refID,
		ReferenceType:  refType,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("log notification: %w", err)
	}

	payload := fonntePayload{
		Target:      phone,
		Message:     message,
		CountryCode: "62",
	}
	body, _ := json.Marshal(payload)

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			s.baseURL+"/send", bytes.NewBuffer(body))
		if err != nil {
			_ = s.repo.UpdateStatus(ctx, n.ID, "failed", err.Error())
			return err
		}
		req.Header.Set("Authorization", s.apiToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			// Timeout / network error → retry per PRD §16.1
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					_ = s.repo.UpdateStatus(ctx, n.ID, "failed", ctx.Err().Error())
					return ctx.Err()
				case <-time.After(fonnteRetryDelay):
					continue
				}
			}
			_ = s.repo.UpdateStatus(ctx, n.ID, "failed", err.Error())
			return err
		}

		var fr fonnteResponse
		_ = json.NewDecoder(resp.Body).Decode(&fr)
		resp.Body.Close()

		// HTTP 5xx → also retry
		if resp.StatusCode >= 500 && attempt < maxAttempts {
			lastErr = fmt.Errorf("fonnte 5xx: %d", resp.StatusCode)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(fonnteRetryDelay):
				continue
			}
		}

		status := "sent"
		if !fr.Status {
			status = "failed"
		}
		_ = s.repo.UpdateStatus(ctx, n.ID, status, fr)
		if !fr.Status {
			return fmt.Errorf("fonnte error: %s", fr.Reason)
		}
		return nil
	}
	return lastErr
}

// ─── Template helpers ─────────────────────────────────────────────────────────

func (s *FonnteService) SendLeadsWelcome(ctx context.Context, phone, name, packageName string, leadID string) error {
	msg := fmt.Sprintf(
		"Halo *%s*! 👋\n\nTerima kasih sudah tertarik dengan paket *%s*.\n\n"+
			"Tim konsultan kami akan segera menghubungi Anda dalam waktu kurang dari 1 jam.\n\n"+
			"Salam,\n*Tim Pintour*", name, packageName)
	refType := "lead"
	return s.Send(ctx, phone, name, notification.TypeLeadsWelcome, msg, &leadID, &refType)
}

func (s *FonnteService) SendLeadsNotifSales(ctx context.Context, consultantPhone, consultantName, leadsName, leadsPhone, packageName, leadID string) error {
	msg := fmt.Sprintf(
		"🔔 *Leads Baru Masuk!*\n\n"+
			"Nama: *%s*\n"+
			"No. WA: *%s*\n"+
			"Paket: *%s*\n\n"+
			"Segera hubungi leads ini. ID: %s", leadsName, leadsPhone, packageName, leadID)
	refType := "lead"
	return s.Send(ctx, consultantPhone, consultantName, notification.TypeLeadsNotifSales, msg, &leadID, &refType)
}

func (s *FonnteService) SendInvoice(ctx context.Context, phone, name, invoiceNumber, amount, dueDate, pdfLink, invoiceID string) error {
	msg := fmt.Sprintf(
		"🧾 *Invoice Pintour*\n\n"+
			"Halo *%s*, invoice Anda telah diterbitkan.\n\n"+
			"No. Invoice: *%s*\n"+
			"Total: *Rp %s*\n"+
			"Jatuh Tempo: *%s*\n\n"+
			"📄 Unduh Invoice: %s\n\n"+
			"Harap lakukan pembayaran sebelum jatuh tempo.",
		name, invoiceNumber, amount, dueDate, pdfLink)
	refType := "invoice"
	return s.Send(ctx, phone, name, notification.TypeInvoiceSent, msg, &invoiceID, &refType)
}

// SendPaymentReminder chases one unpaid invoice. days is the invoice's age, and
// decides both how the message reads and which template it is logged as — taking
// the age rather than a rendered label keeps those two from disagreeing, which
// is how the day-6 reminder came to be recorded as a day-1 one.
func (s *FonnteService) SendPaymentReminder(ctx context.Context, phone, name, invoiceNumber string, days int, invoiceID string) error {
	msg := fmt.Sprintf(
		"⚠️ *Pengingat Pembayaran*\n\n"+
			"Halo *%s*, invoice *%s* Anda belum dibayar (%s sejak diterbitkan).\n\n"+
			"Segera lakukan pembayaran agar proses perjalanan Anda dapat berjalan lancar.\n\n"+
			"Hubungi kami jika ada pertanyaan.", name, invoiceNumber, notification.PaymentReminderLabel(days))
	refType := "invoice"
	return s.Send(ctx, phone, name, notification.PaymentReminderType(days), msg, &invoiceID, &refType)
}

func (s *FonnteService) SendDocRequest(ctx context.Context, phone, name, packageName, portalLink, participantID string) error {
	msg := fmt.Sprintf(
		"✅ *Pembayaran Terkonfirmasi!*\n\n"+
			"Halo *%s*, pembayaran Anda untuk paket *%s* telah dikonfirmasi.\n\n"+
			"Langkah selanjutnya, silakan upload dokumen yang diperlukan melalui:\n"+
			"🔗 %s\n\n"+
			"Username: nomor WA Anda\n"+
			// The password went out in its own message when the account was
			// created (PORTAL_CREDENTIALS). This used to promise a separate
			// message that nothing ever sent.
			"Password: sesuai pesan *Akun Portal Pintour Anda* yang kami kirim sebelumnya\n\n"+
			"Harap lengkapi sebelum H-14 keberangkatan.", name, packageName, portalLink)
	refType := "participant"
	return s.Send(ctx, phone, name, notification.TypeDocRequest, msg, &participantID, &refType)
}

// SendDocRejected notifies participant when a document is rejected (DOC_REJECTED).
func (s *FonnteService) SendDocRejected(ctx context.Context, phone, name, docType, reason, portalLink, participantID string) error {
	msg := fmt.Sprintf(
		"❌ *Dokumen Ditolak*\n\n"+
			"Halo *%s*, dokumen *%s* yang Anda upload memerlukan perbaikan.\n\n"+
			"Alasan: %s\n\n"+
			"Silakan upload ulang dokumen melalui portal:\n"+
			"🔗 %s\n\n"+
			"Hubungi admin jika ada pertanyaan.",
		name, docType, reason, portalLink)
	refType := "document"
	return s.Send(ctx, phone, name, notification.TypeDocRejected, msg, &participantID, &refType)
}

func (s *FonnteService) SendDepartureReminder(ctx context.Context, phone, name, packageName, departureDate, dayLabel, msgType, participantID string) error {
	msg := fmt.Sprintf(
		"✈️ *Reminder Keberangkatan — %s*\n\n"+
			"Halo *%s*! Perjalanan *%s* Anda akan berangkat pada *%s*.\n\n"+
			"Pastikan semua persiapan sudah lengkap. Cek portal peserta Anda untuk informasi terkini.",
		dayLabel, name, packageName, departureDate)
	refType := "participant"
	return s.Send(ctx, phone, name, msgType, msg, &participantID, &refType)
}

// ─── Automasi templates (prompt §3.1) ─────────────────────────────────────────
// Adapted to the existing password-based portal: portal access is via WA + password
// login at {portalLink}, not a one-time token link.

// SendPortalCredentials hands a newly converted participant the temporary
// password for their portal account (FR-PORTAL-01).
//
// The portal has no password-reset flow, so this message is the only copy of the
// credential that ever leaves the system — the raw password is hashed the moment
// it is generated and is never stored anywhere it could be read back.
func (s *FonnteService) SendPortalCredentials(ctx context.Context, phone, name, password, portalLink, participantID string) error {
	refType := "participant"
	return s.Send(ctx, phone, name, notification.TypePortalCredentials,
		portalCredentialsMessage(name, phone, password, portalLink), &participantID, &refType)
}

// portalCredentialsMessage renders the credential message.
//
// It is the one template split out from its Send wrapper, because it is the one
// whose content is the feature: a message that loses the password still sends
// successfully and still logs as delivered, and the participant it was for can
// never get in. Rendering it separately is what lets that be asserted without a
// gateway to send through.
func portalCredentialsMessage(name, phone, password, portalLink string) string {
	return fmt.Sprintf(
		"🔐 *Akun Portal Pintour Anda*\n\n"+
			"Halo *%s*, akun portal peserta Anda sudah dibuat.\n\n"+
			"🔗 Portal: %s\n"+
			"Username: *%s* (nomor WA ini)\n"+
			"Password sementara: *%s*\n\n"+
			"Simpan pesan ini baik-baik dan jangan bagikan ke siapa pun. "+
			"Gunakan portal untuk mengunggah dokumen dan memantau status keberangkatan Anda.",
		name, portalLink, phone, password)
}

// SendPortalActivated notifies a participant that their portal is active after
// payment confirmation (PORTAL_ACTIVATED).
func (s *FonnteService) SendPortalActivated(ctx context.Context, phone, name, email, portalLink, participantID string) error {
	msg := fmt.Sprintf(
		"🎉 *Portal Peserta Aktif!*\n\n"+
			"Halo *%s*! Pembayaran Anda telah dikonfirmasi dan portal peserta Anda kini aktif.\n\n"+
			"🔗 Akses portal: %s\n"+
			"Username: *%s* (nomor WA Anda)\n\n"+
			"Gunakan portal untuk upload dokumen perjalanan dan memantau status keberangkatan Anda.",
		name, portalLink, email)
	refType := "participant"
	return s.Send(ctx, phone, name, notification.TypePortalActivated, msg, &participantID, &refType)
}

// SendPaymentReceived confirms a (partial or full) payment was received (PAYMENT_RECEIVED).
func (s *FonnteService) SendPaymentReceived(ctx context.Context, phone, name, amount, packageName, invoiceID string) error {
	msg := fmt.Sprintf(
		"✅ *Pembayaran Diterima*\n\n"+
			"Halo *%s*, pembayaran Anda sebesar *Rp %s* untuk paket *%s* telah kami terima dan dikonfirmasi.\n\n"+
			"Terima kasih! 🙏",
		name, amount, packageName)
	refType := "invoice"
	return s.Send(ctx, phone, name, notification.TypePaymentReceived, msg, &invoiceID, &refType)
}

// SendPaymentRejected notifies a participant their payment proof was rejected (PAYMENT_REJECTED).
func (s *FonnteService) SendPaymentRejected(ctx context.Context, phone, name, reason, portalLink, invoiceID string) error {
	msg := fmt.Sprintf(
		"❌ *Bukti Pembayaran Ditolak*\n\n"+
			"Halo *%s*, maaf bukti pembayaran Anda ditolak.\n\n"+
			"Alasan: %s\n\n"+
			"Mohon upload ulang bukti pembayaran yang valid melalui portal:\n🔗 %s",
		name, reason, portalLink)
	refType := "invoice"
	return s.Send(ctx, phone, name, notification.TypePaymentRejected, msg, &invoiceID, &refType)
}

// SendPaymentOverdue notifies a participant their invoice passed its due date (PAYMENT_OVERDUE).
func (s *FonnteService) SendPaymentOverdue(ctx context.Context, phone, name, invoiceNumber, adminContact, invoiceID string) error {
	msg := fmt.Sprintf(
		"⚠️ *Invoice Jatuh Tempo*\n\n"+
			"Invoice *#%s* Anda telah melewati batas waktu pembayaran.\n\n"+
			"Segera hubungi tim kami untuk informasi lebih lanjut.\n"+
			"Kontak: %s",
		invoiceNumber, adminContact)
	refType := "invoice"
	return s.Send(ctx, phone, name, notification.TypePaymentOverdue, msg, &invoiceID, &refType)
}

// SendDocApproved notifies a participant all their documents are approved (DOC_APPROVED).
func (s *FonnteService) SendDocApproved(ctx context.Context, phone, name, participantID string) error {
	msg := fmt.Sprintf(
		"✅ *Dokumen Disetujui*\n\n"+
			"Halo *%s*, semua dokumen perjalanan Anda telah diverifikasi dan disetujui!\n\n"+
			"Persiapkan diri untuk keberangkatan. Kami akan mengirim briefing H-14. 🌍",
		name)
	refType := "participant"
	return s.Send(ctx, phone, name, notification.TypeDocApproved, msg, &participantID, &refType)
}

// SendBriefingActivated notifies a participant the digital briefing is active (BRIEFING_ACTIVATED).
func (s *FonnteService) SendBriefingActivated(ctx context.Context, phone, name, tourLeaderName, departureDate, portalLink, participantID string) error {
	msg := fmt.Sprintf(
		"📋 *Briefing Perjalanan Aktif!*\n\n"+
			"Halo *%s*, briefing perjalanan Anda sudah aktif.\n\n"+
			"🔗 Akses di portal: %s\n"+
			"Tour Leader: *%s*\n"+
			"Keberangkatan: *%s*",
		name, portalLink, tourLeaderName, departureDate)
	refType := "participant"
	return s.Send(ctx, phone, name, notification.TypeBriefingActivated, msg, &participantID, &refType)
}

// SendLeadsStale reminds a consultant about an unresponded lead (LEADS_STALE, prompt §2.1).
func (s *FonnteService) SendLeadsStale(ctx context.Context, consultantPhone, consultantName, leadsName, leadID string) error {
	msg := fmt.Sprintf(
		"⚠️ *Leads Belum Direspons*\n\n"+
			"Leads *%s* belum direspons selama 24 jam. Segera hubungi!\n\nID: %s",
		leadsName, leadID)
	refType := "lead"
	return s.Send(ctx, consultantPhone, consultantName, notification.TypeLeadsStale, msg, &leadID, &refType)
}
