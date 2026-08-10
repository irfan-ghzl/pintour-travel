package httpdelivery

// The public group carries no JWT middleware at all, so "no credentials" is the
// expected state rather than a rejection. These prove the harness reaches those
// routes too — every later ticket that validates a public payload starts here.

import (
	"net/http"
	"testing"
)

func TestPublicRoutes_ServedWithoutCredentials(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.anonymous().GET("/api/v1/packages").expectCode(http.StatusOK)
	h.anonymous().GET("/api/v1/packages/umroh-reguler-9-hari").expectCode(http.StatusOK)
	h.anonymous().GET("/health").expectCode(http.StatusOK)
}

// Webhooks are public but carry their own gate: Fonnte's inbound hook is
// authenticated by a shared token rather than a JWT.
func TestFonnteWebhook_RejectsWrongToken(t *testing.T) {
	h := newHarness(t, withChatbotToken("shared-secret"))
	body := map[string]string{"sender": "628123456789", "message": "halo"}

	h.anonymous().POST("/api/v1/webhooks/fonnte?token=wrong", body).
		expectCode(http.StatusUnauthorized)
	h.anonymous().POST("/api/v1/webhooks/fonnte", body).
		expectCode(http.StatusUnauthorized)
	h.anonymous().POST("/api/v1/webhooks/fonnte?token=shared-secret", body).
		expectCode(http.StatusOK)
}
