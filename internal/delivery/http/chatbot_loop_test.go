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

	// Payload sesungguhnya, disalin dari yang dikirim Fonnte saat satu pesan uji
	// dikirimkan. Perhatikan `sender` berisi nomor LAWAN BICARA, bukan nomor
	// perangkat kita — itulah sebabnya membandingkan `device` dengan `sender`
	// tidak pernah cocok, dan penjagaan versi pertama diam-diam tidak berfungsi.
	body := `{"device":"082121952655","sender":"6287789509545",` +
		`"message":"Halo! Terima kasih sudah menghubungi Pintour.\n\n> _Sent via fonnte.com_",` +
		`"name":null,"pushname":null,"isgroup":false,"type":"text"}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/fonnte", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	if err := h.HandleFonnteWebhook(echo.New().NewContext(req, rec)); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "self_echo_ignored") {
		t.Fatalf("gema pesan keluar tidak diabaikan: %s", rec.Body.String())
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

// Webhook status pengiriman datang lewat URL yang sama dan tanpa field message.
func TestFonnteWebhookIgnoresDeliveryStatus(t *testing.T) {
	h := &ChatbotHandler{}

	body := `{"device":"082121952655","id":"172662621","state":"delivered","status":"sent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/fonnte", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	if err := h.HandleFonnteWebhook(echo.New().NewContext(req, rec)); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "ignored") {
		t.Fatalf("webhook status pengiriman tidak diabaikan: %s", rec.Body.String())
	}
}
