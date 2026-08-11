package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// defaultResendBaseURL is where transactional email goes in production.
const defaultResendBaseURL = "https://api.resend.com"

// EmailService sends transactional emails via Resend API.
type EmailService struct {
	apiKey   string
	fromAddr string
	baseURL  string
	client   *http.Client
}

// EmailOption customizes an EmailService at construction.
type EmailOption func(*EmailService)

// WithEmailBaseURL points the service at a different host. Its only use is a
// test server: it is what lets the eighteen §17.1 templates be checked for the
// subject and the content they actually send, instead of only for the guard
// that returns early when no API key is configured. Same shape as
// WithFonnteBaseURL and WithMidtransBaseURL.
func WithEmailBaseURL(baseURL string) EmailOption {
	return func(s *EmailService) {
		if baseURL != "" {
			s.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func NewEmailService(apiKey, fromAddr string, opts ...EmailOption) *EmailService {
	s := &EmailService{
		apiKey:   apiKey,
		fromAddr: fromAddr,
		baseURL:  defaultResendBaseURL,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// Send sends an email via Resend.
//
// Kegagalan dicatat di sini, bukan diserahkan ke pemanggil. Kesepuluh pemanggil
// membuang nilai baliknya dengan `_ =` — masuk akal, karena tidak ada yang mau
// reset password gagal hanya sebab surelnya tidak terkirim. Tapi akibatnya
// seluruh kegagalan surel menjadi tak terlihat: endpoint tetap menjawab 200,
// peserta tidak menerima apa pun, dan tidak ada satu baris pun yang mencatatnya.
//
// Ditemukan saat UAT: Resend menolak 403 karena MAIL_FROM masih memakai alamat
// uji onboarding@resend.dev, yang hanya boleh mengirim ke pemilik akun. Seluruh
// surel ke peserta gagal, dan tidak ada cara mengetahuinya dari dalam sistem.
func (s *EmailService) Send(ctx context.Context, to, subject, htmlBody string) error {
	err := s.send(ctx, to, subject, htmlBody)
	if err != nil {
		log.Printf("kirim surel ke %s gagal (%q): %v", to, subject, err)
	}
	return err
}

func (s *EmailService) send(ctx context.Context, to, subject, htmlBody string) error {
	if s.apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured")
	}
	payload := resendPayload{
		From:    s.fromAddr,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/emails", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		// Badan respons ikut dibawa. Resend menjelaskan sebabnya di sana, dan
		// "status 403" sendirian tidak memberi tahu apa pun yang bisa
		// ditindaklanjuti — sementara pesannya menyebut persis domain mana yang
		// perlu diverifikasi.
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend API error: status %d: %s", resp.StatusCode, bytes.TrimSpace(detail))
	}
	return nil
}

// SendResetPassword sends a password reset email.
func (s *EmailService) SendResetPassword(ctx context.Context, toEmail, toName, resetLink string) error {
	subject := "Reset Password - Pintour Admin"
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:20px">
  <div style="background:#10643b;padding:20px;border-radius:8px;text-align:center">
    <h1 style="color:white;margin:0">Pintour Travel</h1>
  </div>
  <div style="padding:30px 0">
    <h2>Reset Password</h2>
    <p>Halo <strong>%s</strong>,</p>
    <p>Kami menerima permintaan reset password untuk akun Anda.</p>
    <p>Klik tombol di bawah untuk membuat password baru:</p>
    <div style="text-align:center;margin:30px 0">
      <a href="%s" style="background:#10643b;color:white;padding:12px 24px;
         border-radius:6px;text-decoration:none;font-weight:bold">
        Reset Password
      </a>
    </div>
    <p style="color:#888;font-size:12px">
      Link ini berlaku selama 1 jam. Jika Anda tidak meminta reset password, abaikan email ini.
    </p>
  </div>
  <div style="border-top:1px solid #eee;padding-top:15px;text-align:center;color:#888;font-size:12px">
    © %s Pintour Travel
  </div>
</body>
</html>`, toName, resetLink, "2026")
	return s.Send(ctx, toEmail, subject, html)
}
