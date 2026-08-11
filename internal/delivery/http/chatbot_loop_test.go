package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// Loop gema pernah terjadi sungguhan: Fonnte meneruskan pesan yang dikirim
// perangkat kita sendiri kembali ke webhook, bot membalasnya, dan balasan itu
// masuk lagi. Tiga puluh lima balasan dalam dua menit, berhenti hanya karena
// tunnel dimatikan dengan tangan.
//
// Dua test di bawah menjaga kedua lapisan yang sekarang mencegahnya.

func TestFonnteWebhookIgnoresItsOwnEcho(t *testing.T) {
	h := &ChatbotHandler{} // svc nil — permintaan harus ditolak sebelum menyentuhnya

	form := url.Values{
		"device":  {"082121952655"},
		"sender":  {"6282121952655"}, // nomor yang sama, format berbeda
		"message": {"Halo! Terima kasih sudah menghubungi Pintour."},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/fonnte",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	if err := h.HandleFonnteWebhook(echo.New().NewContext(req, rec)); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "self_echo_ignored") {
		t.Fatalf("gema dari perangkat sendiri tidak diabaikan: %s", rec.Body.String())
	}
}

func TestFonnteWebhookStopsARunawayLoop(t *testing.T) {
	// Penjaga ini punya keadaan tingkat paket. Dipakai kunci unik supaya test ini
	// tidak saling mempengaruhi dengan test lain di paket yang sama.
	phone := "628" + time.Now().Format("150405.000000")

	var lastBody string
	for i := 1; i <= 12; i++ {
		form := url.Values{"sender": {phone}, "message": {"halo"}}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/fonnte",
			strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		h := &ChatbotHandler{}
		if err := h.HandleFonnteWebhook(echo.New().NewContext(req, rec)); err != nil {
			t.Fatalf("permintaan %d: %v", i, err)
		}
		lastBody = rec.Body.String()

		// Delapan pertama lolos penjaga laju; yang membedakan hanya sesudahnya.
		if i <= 8 && strings.Contains(lastBody, "rate_limited") {
			t.Fatalf("permintaan ke-%d sudah dibatasi, padahal masih dalam batas wajar", i)
		}
	}

	if !strings.Contains(lastBody, "rate_limited") {
		t.Fatalf("dua belas pesan dalam sekejap tidak memicu penjaga loop: %s", lastBody)
	}
}
