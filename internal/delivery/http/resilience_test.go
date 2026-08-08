package httpdelivery

// Ticket 02 — runtime resilience, asserted through the HTTP seam: a panic must
// come back as a response rather than a dropped connection, and an unreasonably
// large body must be refused before anything buffers it.

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// captureLog redirects the standard logger for the duration of one test, which
// is where recovered panics are reported.
type logCapture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (c *logCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *logCapture) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func captureLog(t *testing.T) *logCapture {
	t.Helper()
	c := &logCapture{}
	prevWriter, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(c)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
	})
	return c
}

// ─── Panic recovery ───────────────────────────────────────────────────────────

// A panic raised deep inside a request — here in the repository, the way a nil
// map write or a bad type assertion would raise one — has to reach the client
// as the API's own error envelope.
func TestPanicInRequestReturnsStructuredError(t *testing.T) {
	c := captureLog(t)
	h := newHarness(t)
	h.seedBaseline()
	h.Leads.PanicOnCreate("repository meledak")

	res := h.anonymous().POST("/api/v1/leads", map[string]string{
		"name": "Prospek Panik", "phone": "081234567890", "package_id": "package-1",
	})

	res.expectCode(http.StatusInternalServerError)
	var body struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	res.decode(&body)
	if body.Success || body.Error == "" || body.Message == "" {
		t.Errorf("body = %s, want the API error envelope", res.Body)
	}
	if strings.Contains(body.Message, "meledak") {
		t.Errorf("message = %q, want no internal detail disclosed to the client", body.Message)
	}

	logged := c.text()
	if !strings.Contains(logged, "meledak") {
		t.Errorf("log = %q, want the panic value recorded", logged)
	}
	if !strings.Contains(logged, "/api/v1/leads") {
		t.Errorf("log = %q, want the route that panicked", logged)
	}
	if !strings.Contains(logged, "lead_handler.go") {
		t.Errorf("log = %q, want a stack trace reaching the handler", logged)
	}
}

// The server has to keep serving afterwards — the point of recovering is that
// one bad request costs one request.
func TestServerKeepsServingAfterAPanic(t *testing.T) {
	captureLog(t)
	h := newHarness(t)
	h.seedBaseline()
	h.Leads.PanicOnCreate("repository meledak")

	h.anonymous().POST("/api/v1/leads", map[string]string{
		"name": "Prospek Panik", "phone": "081234567890", "package_id": "package-1",
	}).expectCode(http.StatusInternalServerError)

	h.anonymous().GET("/api/v1/packages").expectCode(http.StatusOK)
}

// The notifications a request fires off run on their own goroutine, which the
// HTTP server does not cover: an unrecovered panic there ends the process — and
// this test binary with it — rather than the one notification. Creating a lead
// answers the caller first and notifies admins afterwards, so a panic in the
// admin lookup lands squarely in that background work.
func TestPanicInBackgroundWorkDoesNotKillTheProcess(t *testing.T) {
	c := captureLog(t)
	h := newHarness(t)
	h.seedBaseline()
	h.Users.PanicOnListByRole("admin", "notifikasi meledak")

	h.anonymous().POST("/api/v1/leads", map[string]string{
		"name": "Prospek Dua", "phone": "081234567891", "package_id": "package-1",
	}).expectCode(http.StatusCreated)

	logged := awaitLogged(t, c, "notifikasi meledak")
	if !strings.Contains(logged, "notifikasi lead baru") {
		t.Errorf("log = %q, want the label of the background job that panicked", logged)
	}

	// Still serving: the panic cost one notification, not the process.
	h.anonymous().GET("/api/v1/packages").expectCode(http.StatusOK)
}

// awaitLogged waits for want to appear in the captured log, which is how a test
// synchronises with work that outlives the request that started it.
func awaitLogged(t *testing.T, c *logCapture, want string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		logged := c.text()
		if strings.Contains(logged, want) {
			return logged
		}
		if time.Now().After(deadline) {
			t.Fatalf("log = %q, want it to contain %q within 2s", logged, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// ─── Body size limit ──────────────────────────────────────────────────────────

// Every public endpoint — the two logins, the leads form, both webhooks — has
// to refuse a body no legitimate client would send.
func TestOversizedBodyRejectedOnPublicEndpoints(t *testing.T) {
	h := newHarness(t, withChatbotToken("shared-secret"))
	h.seedBaseline()

	oversized := strings.Repeat("a", 2<<20) // 2 MB of JSON string
	payload := `{"padding":"` + oversized + `"}`

	for _, path := range []string{
		"/api/v1/leads",
		"/api/v1/auth/login",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
		"/api/v1/portal/login",
		"/api/v1/webhooks/midtrans",
		"/api/v1/webhooks/fonnte?token=shared-secret",
	} {
		h.anonymous().POST(path, payload).expectCode(http.StatusRequestEntityTooLarge)
	}
}

// The limit must not get in the way of the requests those endpoints exist for.
func TestNormalSizedBodyStillAccepted(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	// Wrong credentials, but the request was read: the point is that it reached
	// the handler instead of being refused for its size.
	res := h.anonymous().POST("/api/v1/auth/login", map[string]string{
		"email": "admin@pintour.test", "password": "salah",
	})
	if res.Code == http.StatusRequestEntityTooLarge {
		t.Errorf("a normal login was rejected as too large")
	}
}

// A body that never declares its size (chunked transfer encoding) must be
// refused as well, and refused without reading it to the end — otherwise the
// limit is a formality an attacker skips by omitting Content-Length.
func TestOversizedBodyWithoutContentLengthIsRefusedEarly(t *testing.T) {
	h := newHarness(t)

	body := &countingReader{remaining: 64 << 20} // 64 MB, never fully sent
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads", body)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = -1
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.e.ServeHTTP(rec, req)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not finish: the body is being read without a ceiling")
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusRequestEntityTooLarge, rec.Body)
	}
	if read := body.read(); read > maxJSONBody+1024 {
		t.Errorf("read %d bytes of the body, want the request refused near the %d-byte limit", read, maxJSONBody)
	}
}

// countingReader yields an endless run of bytes and records how many were taken,
// standing in for a client that streams more than it is allowed to.
type countingReader struct {
	mu        sync.Mutex
	remaining int64
	taken     int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > r.remaining {
		n = r.remaining
	}
	for i := int64(0); i < n; i++ {
		p[i] = 'a'
	}
	r.remaining -= n
	r.taken += n
	return int(n), nil
}

func (r *countingReader) read() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.taken
}

// File uploads are the one case where several megabytes is a normal body, so
// the limit has to let a legitimate document through. The harness has no
// storage configured, so the handler answers 503 — which still proves the
// request was not turned away for its size.
func TestUploadOfALegitimateFileIsNotRefusedForItsSize(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	part, err := form.CreateFormFile("file", "paspor.jpg")
	if err != nil {
		t.Fatalf("build multipart body: %v", err)
	}
	if _, err := part.Write(make([]byte, 5<<20)); err != nil { // 5 MB, the §16.2 ceiling
		t.Fatalf("write multipart file: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/upload/document", body)
	req.Header.Set(echo.HeaderContentType, form.FormDataContentType())
	req.Header.Set("X-Portal-Token", h.portalToken("participant-1", ""))
	rec := httptest.NewRecorder()
	h.e.ServeHTTP(rec, req)

	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Errorf("a 5 MB document upload was refused as too large")
	}
}

// The ceiling is not only for public routes: an authenticated staff endpoint
// that takes JSON is capped by the same middleware.
func TestOversizedJSONRejectedOnStaffEndpoints(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	payload := `{"padding":"` + strings.Repeat("a", 2<<20) + `"}`
	h.as("super_admin").POST("/api/v1/admin/users", payload).
		expectCode(http.StatusRequestEntityTooLarge)
}

// ─── Password reset tokens ────────────────────────────────────────────────────

// A reset token nobody redeems must not stay in memory forever: the store is
// swept, and only the entries that are past their expiry go.
func TestSweepDropsOnlyExpiredResetTokens(t *testing.T) {
	now := time.Now()
	store := newResetTokenStore()
	store.issue("kadaluarsa", "lama@pintour.test", now.Add(-time.Minute))
	store.issue("masih-hidup", "baru@pintour.test", now.Add(time.Hour))

	if dropped := store.sweep(now); dropped != 1 {
		t.Errorf("sweep dropped %d entries, want 1", dropped)
	}
	if size := store.size(); size != 1 {
		t.Errorf("size after sweep = %d, want 1", size)
	}
	if _, ok := store.consume("masih-hidup", now); !ok {
		t.Error("the unexpired token was swept away with the expired one")
	}
}

// Redeeming a token consumes it, so a reset link cannot be replayed — and an
// expired one is refused even if the sweeper has not come round yet.
func TestResetTokenIsRedeemableOnce(t *testing.T) {
	now := time.Now()
	store := newResetTokenStore()
	store.issue("sekali", "staf@pintour.test", now.Add(time.Hour))

	entry, ok := store.consume("sekali", now)
	if !ok || entry.email != "staf@pintour.test" {
		t.Fatalf("consume = %+v, %v; want the issued entry", entry, ok)
	}
	if _, ok := store.consume("sekali", now); ok {
		t.Error("the same reset token was accepted twice")
	}

	store.issue("basi", "staf@pintour.test", now.Add(-time.Second))
	if _, ok := store.consume("basi", now); ok {
		t.Error("an expired reset token was accepted")
	}
	if size := store.size(); size != 0 {
		t.Errorf("size = %d, want the expired entry dropped on the way out", size)
	}
}

// Repeated forgot-password requests each mint a token. What keeps that from
// growing the process without bound is the sweep, so this asserts the whole
// path: requests in, entries stored, sweep, nothing left.
func TestRepeatedForgotPasswordDoesNotGrowMemoryForever(t *testing.T) {
	h := newHarness(t)
	before := resetTokens.size()

	const requests = 5
	for i := 0; i < requests; i++ {
		h.anonymous().POST("/api/v1/auth/forgot-password", map[string]string{
			"email": "admin@pintour.test",
		}).expectCode(http.StatusOK)
	}
	if grew := resetTokens.size() - before; grew != requests {
		t.Fatalf("store grew by %d, want one entry per request (%d)", grew, requests)
	}

	resetTokens.sweep(time.Now().Add(2 * resetTokenTTL))
	if size := resetTokens.size(); size != 0 {
		t.Errorf("size after sweep = %d, want every expired token dropped", size)
	}
}

// The eviction has to happen on its own, without a request to trigger it — a
// store swept only when someone asks is not swept at all overnight.
func TestSweeperEvictsWithoutBeingAsked(t *testing.T) {
	store := newResetTokenStore()
	store.issue("kadaluarsa", "lama@pintour.test", time.Now().Add(-time.Minute))
	store.issue("masih-hidup", "baru@pintour.test", time.Now().Add(time.Hour))

	store.startSweeper(time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)
	for store.size() > 1 {
		if time.Now().After(deadline) {
			t.Fatalf("size = %d after 2s, want the expired token evicted by the sweeper alone", store.size())
		}
		time.Sleep(time.Millisecond)
	}
	if _, ok := store.consume("masih-hidup", time.Now()); !ok {
		t.Error("the sweeper took the unexpired token too")
	}
}
