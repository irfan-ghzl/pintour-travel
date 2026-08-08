package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
)

// FonnteService sends WhatsApp messages via Fonnte API.
type FonnteService struct {
	apiToken string
	repo     notification.Repository
	client   *http.Client
}

func NewFonnteService(apiToken string, repo notification.Repository) *FonnteService {
	return &FonnteService{
		apiToken: apiToken,
		repo:     repo,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
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
			"https://api.fonnte.com/send", bytes.NewBuffer(body))
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

func (s *FonnteService) SendPaymentReminder(ctx context.Context, phone, name, invoiceNumber, dayLabel, invoiceID string) error {
	msg := fmt.Sprintf(
		"⚠️ *Pengingat Pembayaran*\n\n"+
			"Halo *%s*, invoice *%s* Anda belum dibayar (%s sejak diterbitkan).\n\n"+
			"Segera lakukan pembayaran agar proses perjalanan Anda dapat berjalan lancar.\n\n"+
			"Hubungi kami jika ada pertanyaan.", name, invoiceNumber, dayLabel)
	msgType := notification.TypePaymentReminder1
	if dayLabel == "3 hari" {
		msgType = notification.TypePaymentReminder3
	}
	refType := "invoice"
	return s.Send(ctx, phone, name, msgType, msg, &invoiceID, &refType)
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
