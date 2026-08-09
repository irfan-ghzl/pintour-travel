package service

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"
)

// This file implements the participant + admin email templates from the
// automation spec (§3.2 / §3.3). EmailService stays a dumb sender — callers
// fetch the data and pass primitives, consistent with the clean-arch layering.

// emailLayout wraps body HTML in the shared Pintour email shell.
func emailLayout(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#222">
  <div style="background:#10643b;padding:20px;border-radius:8px;text-align:center">
    <h1 style="color:white;margin:0">Pintour Travel</h1>
  </div>
  <div style="padding:30px 0">
    <h2 style="color:#10643b">%s</h2>
    %s
  </div>
  <div style="border-top:1px solid #eee;padding-top:15px;text-align:center;color:#888;font-size:12px">
    © %s Pintour Travel · Digenerate oleh sistem Pintour
  </div>
</body>
</html>`, html.EscapeString(title), body, time.Now().Format("2006"))
}

func btn(href, label string) string {
	return fmt.Sprintf(`<div style="text-align:center;margin:30px 0">
      <a href="%s" style="background:#10643b;color:white;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold">%s</a>
    </div>`, html.EscapeString(href), html.EscapeString(label))
}

// OverdueInvoiceRow is one line in the admin overdue-invoice digest (§3.3).
type OverdueInvoiceRow struct {
	ParticipantName string
	InvoiceNumber   string
	Amount          string
}

// ─── Participant emails (§3.2) ────────────────────────────────────────────────

// SendEmailInvoice sends a formal invoice email with a payment-detail table.
func (s *EmailService) SendEmailInvoice(ctx context.Context, to, name, invoiceNumber, packageName, amount, dueDate, portalLink string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Invoice perjalanan Anda telah diterbitkan. Berikut rinciannya:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">No. Invoice</td><td style="padding:8px;border:1px solid #eee"><strong>%s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Paket</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Total</td><td style="padding:8px;border:1px solid #eee"><strong>Rp %s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Jatuh Tempo</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
    </table>
    %s`,
		html.EscapeString(name), html.EscapeString(invoiceNumber), html.EscapeString(packageName),
		html.EscapeString(amount), html.EscapeString(dueDate), btn(portalLink, "Buka Portal & Bayar"))
	return s.Send(ctx, to, fmt.Sprintf("Invoice Perjalanan #%s - Pintour", invoiceNumber),
		emailLayout("Invoice Perjalanan", body))
}

// SendEmailPaymentReceived sends a digital receipt.
func (s *EmailService) SendEmailPaymentReceived(ctx context.Context, to, name, invoiceNumber, amount, paidDate string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Pembayaran Anda telah kami terima dan dikonfirmasi. Terima kasih!</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">No. Invoice</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Nominal</td><td style="padding:8px;border:1px solid #eee"><strong>Rp %s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Tanggal</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
    </table>`,
		html.EscapeString(name), html.EscapeString(invoiceNumber), html.EscapeString(amount), html.EscapeString(paidDate))
	return s.Send(ctx, to, fmt.Sprintf("Konfirmasi Pembayaran Diterima - #%s", invoiceNumber),
		emailLayout("Pembayaran Diterima", body))
}

// SendEmailPaymentRejected explains why a payment proof was rejected.
func (s *EmailService) SendEmailPaymentRejected(ctx context.Context, to, name, invoiceNumber, reason, portalLink string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Bukti pembayaran Anda untuk invoice <strong>#%s</strong> perlu diperbaiki.</p>
    <p style="background:#fde8e8;padding:12px;border-radius:6px"><strong>Alasan:</strong> %s</p>
    <p><strong>Cara upload ulang:</strong></p>
    <ol><li>Buka portal peserta.</li><li>Pilih invoice terkait.</li><li>Upload ulang bukti transfer yang jelas dan valid.</li></ol>
    %s`,
		html.EscapeString(name), html.EscapeString(invoiceNumber), html.EscapeString(reason), btn(portalLink, "Upload Ulang"))
	return s.Send(ctx, to, fmt.Sprintf("Bukti Pembayaran Perlu Diperbaiki - #%s", invoiceNumber),
		emailLayout("Bukti Pembayaran Perlu Diperbaiki", body))
}

// SendEmailPaymentOverdue is a formal overdue notice with admin contact.
func (s *EmailService) SendEmailPaymentOverdue(ctx context.Context, to, name, invoiceNumber, adminContact string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Invoice <strong>#%s</strong> Anda telah melewati batas waktu pembayaran (jatuh tempo).</p>
    <p>Mohon segera selesaikan pembayaran atau hubungi tim kami untuk penyelesaian.</p>
    <p>Kontak admin: <strong>%s</strong></p>`,
		html.EscapeString(name), html.EscapeString(invoiceNumber), html.EscapeString(adminContact))
	return s.Send(ctx, to, fmt.Sprintf("⚠️ Invoice #%s Telah Jatuh Tempo", invoiceNumber),
		emailLayout("Invoice Jatuh Tempo", body))
}

// SendEmailDocRequest lists the documents a participant must upload.
func (s *EmailService) SendEmailDocRequest(ctx context.Context, to, name, packageName string, docTypes []string, deadline, portalLink string) error {
	docsHTML := `<p>Daftar dokumen yang dibutuhkan dapat dilihat di portal peserta Anda.</p>`
	if len(docTypes) > 0 {
		rows := strings.Builder{}
		for _, d := range docTypes {
			rows.WriteString(fmt.Sprintf(`<tr><td style="padding:8px;border:1px solid #eee">%s</td></tr>`, html.EscapeString(d)))
		}
		docsHTML = fmt.Sprintf(`<table style="width:100%%;border-collapse:collapse;margin:16px 0">%s</table>`, rows.String())
	}
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Untuk perjalanan <strong>%s</strong>, mohon lengkapi dokumen perjalanan sebelum <strong>%s</strong>:</p>
    %s
    %s`,
		html.EscapeString(name), html.EscapeString(packageName), html.EscapeString(deadline), docsHTML, btn(portalLink, "Upload Dokumen"))
	return s.Send(ctx, to, fmt.Sprintf("Dokumen Perjalanan Diperlukan - %s", packageName),
		emailLayout("Dokumen Perjalanan Diperlukan", body))
}

// SendEmailDocRejected explains a rejected document.
func (s *EmailService) SendEmailDocRejected(ctx context.Context, to, name, documentType, reason, portalLink string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Dokumen <strong>%s</strong> Anda perlu diperbaiki.</p>
    <p style="background:#fde8e8;padding:12px;border-radius:6px"><strong>Alasan:</strong> %s</p>
    %s`,
		html.EscapeString(name), html.EscapeString(documentType), html.EscapeString(reason), btn(portalLink, "Upload Ulang"))
	return s.Send(ctx, to, fmt.Sprintf("Dokumen %s Perlu Diperbaiki", documentType),
		emailLayout("Dokumen Perlu Diperbaiki", body))
}

// SendEmailDocApproved confirms all documents are valid.
func (s *EmailService) SendEmailDocApproved(ctx context.Context, to, name string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Semua dokumen perjalanan Anda telah diverifikasi dan disetujui. ✅</p>
    <p>Langkah berikutnya: persiapkan diri untuk keberangkatan. Kami akan mengirim briefing H-14.</p>`,
		html.EscapeString(name))
	return s.Send(ctx, to, "✅ Dokumen Perjalanan Lengkap & Disetujui",
		emailLayout("Dokumen Lengkap & Disetujui", body))
}

// SendEmailPortalActivated sends portal credentials/login info.
func (s *EmailService) SendEmailPortalActivated(ctx context.Context, to, name, username, portalLink string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Akun portal peserta Anda telah aktif. Gunakan kredensial berikut untuk login:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">Username</td><td style="padding:8px;border:1px solid #eee"><strong>%s</strong> (nomor WA)</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Password</td><td style="padding:8px;border:1px solid #eee">dikirim via WhatsApp</td></tr>
    </table>
    %s`,
		html.EscapeString(name), html.EscapeString(username), btn(portalLink, "Login Portal"))
	return s.Send(ctx, to, "Akun Portal Peserta Anda Telah Aktif - Pintour",
		emailLayout("Portal Peserta Aktif", body))
}

// SendEmailBriefingActivated sends the H-14 briefing pack.
func (s *EmailService) SendEmailBriefingActivated(ctx context.Context, to, name, packageName, tourLeaderName, departureDate, portalLink string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>Briefing perjalanan <strong>%s</strong> sudah tersedia di portal Anda.</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">Tour Leader</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Keberangkatan</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
    </table>
    <p>Di portal Anda akan menemukan itinerary lengkap, aturan perjalanan, dan packing list.</p>
    %s`,
		html.EscapeString(name), html.EscapeString(packageName), html.EscapeString(tourLeaderName),
		html.EscapeString(departureDate), btn(portalLink, "Buka Briefing"))
	return s.Send(ctx, to, fmt.Sprintf("Briefing Perjalanan H-14 - %s", packageName),
		emailLayout("Briefing Perjalanan H-14", body))
}

// SendEmailReminderH14 sends the 2-weeks-to-go checklist.
func (s *EmailService) SendEmailReminderH14(ctx context.Context, to, name, packageName, departureDate, portalLink string) error {
	body := fmt.Sprintf(`<p>Halo <strong>%s</strong>,</p>
    <p>2 minggu lagi keberangkatan <strong>%s</strong> pada <strong>%s</strong>!</p>
    <p><strong>Checklist H-14:</strong></p>
    <ul><li>Pastikan semua dokumen sudah disetujui.</li><li>Cek masa berlaku paspor.</li><li>Siapkan kebutuhan pribadi sesuai packing list.</li></ul>
    %s`,
		html.EscapeString(name), html.EscapeString(packageName), html.EscapeString(departureDate), btn(portalLink, "Buka Portal"))
	return s.Send(ctx, to, fmt.Sprintf("2 Minggu Lagi! Persiapan Keberangkatan %s", packageName),
		emailLayout("Persiapan Keberangkatan H-14", body))
}

// ─── Admin emails (§3.3) ──────────────────────────────────────────────────────

// SendEmailAdminNewLeads notifies an admin of a new lead.
func (s *EmailService) SendEmailAdminNewLeads(ctx context.Context, to, leadName, leadPhone, leadEmail, packageName, crmLink string) error {
	body := fmt.Sprintf(`<p>Leads baru masuk:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">Nama</td><td style="padding:8px;border:1px solid #eee"><strong>%s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">No. WA</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Email</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Paket</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
    </table>
    %s`,
		html.EscapeString(leadName), html.EscapeString(leadPhone), html.EscapeString(leadEmail),
		html.EscapeString(packageName), btn(crmLink, "Buka CRM"))
	return s.Send(ctx, to, fmt.Sprintf("Leads Baru: %s - %s", leadName, packageName),
		emailLayout("Leads Baru", body))
}

// SendEmailAdminPaymentProof notifies an admin a payment proof needs verification.
func (s *EmailService) SendEmailAdminPaymentProof(ctx context.Context, to, participantName, amount, verifyLink string) error {
	body := fmt.Sprintf(`<p>Bukti bayar baru perlu diverifikasi:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">Peserta</td><td style="padding:8px;border:1px solid #eee"><strong>%s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Nominal klaim</td><td style="padding:8px;border:1px solid #eee">Rp %s</td></tr>
    </table>
    %s`,
		html.EscapeString(participantName), html.EscapeString(amount), btn(verifyLink, "Verifikasi"))
	return s.Send(ctx, to, fmt.Sprintf("Bukti Bayar Baru Perlu Diverifikasi - %s", participantName),
		emailLayout("Bukti Bayar Perlu Diverifikasi", body))
}

// SendEmailAdminDocUploaded notifies an admin a new document needs review.
func (s *EmailService) SendEmailAdminDocUploaded(ctx context.Context, to, participantName, documentType, reviewLink string) error {
	body := fmt.Sprintf(`<p>Dokumen baru perlu direview:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">Peserta</td><td style="padding:8px;border:1px solid #eee"><strong>%s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Jenis Dokumen</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
    </table>
    %s`,
		html.EscapeString(participantName), html.EscapeString(documentType), btn(reviewLink, "Review Dokumen"))
	return s.Send(ctx, to, fmt.Sprintf("Dokumen Baru Perlu Direview - %s", participantName),
		emailLayout("Dokumen Perlu Direview", body))
}

// SendEmailAdminPaymentOverdue sends one digest of all invoices that expired today (§2.3).
func (s *EmailService) SendEmailAdminPaymentOverdue(ctx context.Context, to string, rows []OverdueInvoiceRow, actionLink string) error {
	tr := strings.Builder{}
	for _, r := range rows {
		tr.WriteString(fmt.Sprintf(
			`<tr><td style="padding:8px;border:1px solid #eee">%s</td><td style="padding:8px;border:1px solid #eee">%s</td><td style="padding:8px;border:1px solid #eee">Rp %s</td></tr>`,
			html.EscapeString(r.ParticipantName), html.EscapeString(r.InvoiceNumber), html.EscapeString(r.Amount)))
	}
	body := fmt.Sprintf(`<p>%d invoice jatuh tempo hari ini:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr style="background:#f3f4f6"><th style="padding:8px;border:1px solid #eee;text-align:left">Peserta</th><th style="padding:8px;border:1px solid #eee;text-align:left">No. Invoice</th><th style="padding:8px;border:1px solid #eee;text-align:left">Nominal</th></tr>
      %s
    </table>
    %s`, len(rows), tr.String(), btn(actionLink, "Buka Invoice"))
	return s.Send(ctx, to, fmt.Sprintf("⚠️ %d Invoice Jatuh Tempo Hari Ini - Pintour", len(rows)),
		emailLayout("Invoice Jatuh Tempo Hari Ini", body))
}

// SendEmailAdminQuotaWarning warns admins a batch is nearly full (§2.4).
func (s *EmailService) SendEmailAdminQuotaWarning(ctx context.Context, to, packageName, departureDate string, remaining, quota int) error {
	pct := 0
	if quota > 0 {
		pct = (quota - remaining) * 100 / quota
	}
	body := fmt.Sprintf(`<p>Kuota batch hampir penuh:</p>
    <table style="width:100%%;border-collapse:collapse;margin:16px 0">
      <tr><td style="padding:8px;border:1px solid #eee">Paket</td><td style="padding:8px;border:1px solid #eee"><strong>%s</strong></td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Keberangkatan</td><td style="padding:8px;border:1px solid #eee">%s</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Sisa Kursi</td><td style="padding:8px;border:1px solid #eee">%d dari %d</td></tr>
      <tr><td style="padding:8px;border:1px solid #eee">Terisi</td><td style="padding:8px;border:1px solid #eee">%d%%</td></tr>
    </table>`,
		html.EscapeString(packageName), html.EscapeString(departureDate), remaining, quota, pct)
	return s.Send(ctx, to, fmt.Sprintf("⚠️ Kuota Hampir Penuh: %s - %s", packageName, departureDate),
		emailLayout("Kuota Hampir Penuh", body))
}
