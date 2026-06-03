package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EmailService sends transactional emails via Resend API.
type EmailService struct {
	apiKey   string
	fromAddr string
	client   *http.Client
}

func NewEmailService(apiKey, fromAddr string) *EmailService {
	return &EmailService{
		apiKey:   apiKey,
		fromAddr: fromAddr,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// Send sends an email via Resend.
func (s *EmailService) Send(ctx context.Context, to, subject, htmlBody string) error {
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
		"https://api.resend.com/emails", bytes.NewBuffer(body))
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
		return fmt.Errorf("resend API error: status %d", resp.StatusCode)
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
