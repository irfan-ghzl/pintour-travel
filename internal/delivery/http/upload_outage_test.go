package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Storage that is configured but unreachable is a different failure from
// storage that is absent, and for a while it was reported as a third thing
// entirely: 400 UPLOAD_FAILED, carrying the transport error verbatim.
//
// Found live rather than by reading code. The Supabase project named in .env
// had stopped resolving, and the participant's upload dialog showed them
//
//	Post "https://<project>.supabase.co/storage/v1/object/participant-documents/
//	<participant-uuid>/<timestamp>-ktp.pdf": dial tcp: lookup ... no such host
//
// which blames the uploader for an outage, names our bucket host, spells out
// the object path, and leaves nothing in the log for whoever could actually fix
// it. The browser fallback that offers a manual URL watches for 503, so a 400
// also withheld the one route still open to them.

// unreachableStorage returns the address of a server that is not listening:
// started to reserve a real port, then closed. Pointing storage at it produces
// a genuine transport failure rather than a hand-made error, which is the point
// — the mapping under test keys on what the HTTP client actually returns.
func unreachableStorage(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.URL
	srv.Close()
	return addr
}

func TestUpload_AnswersUnavailableWhenTheBucketCannotBeReached(t *testing.T) {
	cases := map[string]struct {
		path   string
		asRole func(h *harness) *client
	}{
		"dokumen peserta": {
			"/api/v1/portal/upload/document",
			func(h *harness) *client { return h.asParticipant("participant-1") },
		},
		"bukti bayar": {
			"/api/v1/portal/upload/payment-proof",
			func(h *harness) *client { return h.asParticipant("participant-1") },
		},
		"gambar paket": {
			"/api/v1/admin/packages/package-1/images/upload",
			func(h *harness) *client { return h.as("admin") },
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, withStorageServer(unreachableStorage(t)))
			h.seedBaseline()

			body, contentType := multipartFile(t, "file", "ktp.pdf", []byte("%PDF-1.4 palsu"))
			res := tc.asRole(h).withHeader("Content-Type", contentType).
				POST(tc.path, body.Bytes())

			res.expectCode(http.StatusServiceUnavailable)

			// The status is what the fallback keys on, but the body is what the
			// participant reads: it must not hand them our bucket host or the
			// object path we tried to write.
			for _, leak := range []string{"dial tcp", "supabase", "storage/v1", "participant-documents"} {
				if strings.Contains(strings.ToLower(string(res.Body)), leak) {
					t.Errorf("badan respons memuat detail internal %q: %s", leak, res.Body)
				}
			}
		})
	}
}

// A file the backend actively rejects is still the caller's to fix, so it must
// keep answering 4xx. Without this, "map the outage to 503" quietly becomes
// "map every upload failure to 503", and a real rejection starts telling the
// participant to wait for a service that is working fine.
func TestUpload_StillRejectsWhatTheBucketRefuses(t *testing.T) {
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"Duplicate","message":"object exists"}`))
	}))
	defer refusing.Close()

	h := newHarness(t, withStorageServer(refusing.URL))
	h.seedBaseline()

	body, contentType := multipartFile(t, "file", "ktp.pdf", []byte("%PDF-1.4 palsu"))
	h.asParticipant("participant-1").withHeader("Content-Type", contentType).
		POST("/api/v1/portal/upload/document", body.Bytes()).
		expectCode(http.StatusBadRequest)
}

// A bucket answering 5xx is the same outage as one that will not accept a
// connection, and gets the same treatment — a paused project is reachable at the
// TCP level long after it has stopped storing anything.
func TestUpload_TreatsBucketServerErrorsAsAnOutage(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer failing.Close()

	h := newHarness(t, withStorageServer(failing.URL))
	h.seedBaseline()

	body, contentType := multipartFile(t, "file", "ktp.pdf", []byte("%PDF-1.4 palsu"))
	h.asParticipant("participant-1").withHeader("Content-Type", contentType).
		POST("/api/v1/portal/upload/document", body.Bytes()).
		expectCode(http.StatusServiceUnavailable)
}
