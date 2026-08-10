package httpdelivery

// Access control (tiket 04).
//
// A signed URL is a bearer capability: for an hour, whoever holds it can read
// the object with no further authentication. So these tests are written from the
// outside in — they assert what a caller can obtain, not how the endpoint
// decides. The sentinel is the stub storage below: an unauthorized request must
// leave it untouched, because a request that reached the signer at all has
// already been granted the capability whatever status is returned afterwards.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// ─── Stub object storage ──────────────────────────────────────────────────────

// signingStorage stands in for Supabase Storage. It signs whatever it is asked
// for and records the object named by each request, so a test can assert not
// only the status the caller saw but whether a file was unlocked at all.
type signingStorage struct {
	*httptest.Server
	mu     sync.Mutex
	signed []string // "bucket/path" per sign request received
}

func newSigningStorage(t *testing.T) *signingStorage {
	t.Helper()
	s := &signingStorage{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		object := strings.TrimPrefix(r.URL.Path, "/storage/v1/object/sign/")
		s.mu.Lock()
		s.signed = append(s.signed, object)
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"signedURL":"/object/sign/%s?token=stub"}`, object)
	}))
	t.Cleanup(s.Close)
	return s
}

// objectsSigned returns every object the server was asked to sign, in order.
func (s *signingStorage) objectsSigned() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.signed...)
}

// expectNothingSigned fails unless the storage was never reached — the assertion
// that matters for a refusal, since the status alone cannot show whether a URL
// was minted before the refusal.
func (s *signingStorage) expectNothingSigned(t *testing.T) {
	t.Helper()
	if got := s.objectsSigned(); len(got) != 0 {
		t.Errorf("storage menandatangani %v; seharusnya tidak ada berkas yang dibuka sama sekali", got)
	}
}

// ─── World ────────────────────────────────────────────────────────────────────

// seedTwoParticipants gives each of two unrelated participants one document and
// one payment proof, so "peserta A membuka berkas peserta B" can be stated
// directly. Participant A is a returning customer: portal user "portal-user-a"
// also owns the earlier tour "participant-a-2019".
func seedTwoParticipants(h *harness) {
	h.PortalUsers.Seed(portaluser.PortalUser{
		ID: "portal-user-a", Phone: "628111000001", Name: "Peserta A",
	})
	lastYear := time.Now().AddDate(0, 0, -365)
	portalUserA := "portal-user-a"

	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-a", PortalUserID: &portalUserA, BatchID: "batch-1",
		Name: "Peserta A", Phone: "628111000001", IsActive: true,
		BatchDepartureDate: ptrTime(time.Now().AddDate(0, 0, 30)),
	})
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-a-2019", PortalUserID: &portalUserA, BatchID: "batch-0",
		Name: "Peserta A", Phone: "628111000001", IsActive: false,
		BatchDepartureDate: &lastYear,
	})
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-b", BatchID: "batch-1",
		Name: "Peserta B", Phone: "628111000002", IsActive: true,
	})

	for _, pid := range []string{"participant-a", "participant-a-2019", "participant-b"} {
		h.Documents.Seed(domainDocument.Document{
			ID: "document-" + pid, ParticipantID: pid, DocumentType: "passport",
			FilePath: pid + "/passport.jpg", FileName: "passport.jpg", Status: "menunggu",
		})
		h.Invoices.Seed(domainInvoice.Invoice{
			ID: "invoice-" + pid, InvoiceNumber: "INV-" + pid, ParticipantID: pid,
			BatchID: "batch-1", Amount: 25000000, Status: "menunggu_bayar",
			DueDate: calendar.Today().AddDays(1),
		})
		h.Proofs.Seed(domainInvoice.PaymentProof{
			ID: "proof-" + pid, InvoiceID: "invoice-" + pid,
			FilePath: pid + "/transfer.jpg", AmountClaimed: 5000000, Status: "menunggu",
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// privateFiles names the two kinds of private file the signer serves, with one
// resource id per owner. Every access-control case runs over both: a rule that
// holds for a passport scan but not for a bank transfer is not a rule.
type privateFileCase struct {
	name  string
	query func(participantID string) string
}

var privateFiles = []privateFileCase{
	{"dokumen", func(pid string) string { return "type=document&id=document-" + pid }},
	{"bukti bayar", func(pid string) string { return "type=payment_proof&id=proof-" + pid }},
}

// ─── Portal path: ownership ───────────────────────────────────────────────────

// The defect this ticket exists for: one participant obtaining a signed URL to
// another participant's passport or bank transfer. 404 rather than 403, so the
// answer does not confirm that the other participant's file exists.
func TestPortalSignedURL_RefusesAnotherParticipantsFile(t *testing.T) {
	for _, file := range privateFiles {
		t.Run(file.name, func(t *testing.T) {
			storage := newSigningStorage(t)
			h := newHarness(t, withStorageServer(storage.URL))
			seedTwoParticipants(h)

			h.asParticipant("participant-a", "portal-user-a").
				GET("/api/v1/portal/signed-url?" + file.query("participant-b")).
				expectCode(http.StatusNotFound)

			storage.expectNothingSigned(t)
		})
	}
}

func TestPortalSignedURL_SignsOwnFile(t *testing.T) {
	for _, file := range privateFiles {
		t.Run(file.name, func(t *testing.T) {
			storage := newSigningStorage(t)
			h := newHarness(t, withStorageServer(storage.URL))
			seedTwoParticipants(h)

			res := h.asParticipant("participant-a", "portal-user-a").
				GET("/api/v1/portal/signed-url?" + file.query("participant-a"))
			res.expectCode(http.StatusOK)

			if url, _ := res.data()["url"].(string); url == "" {
				t.Errorf("respons tidak memuat url: %s", res.Body)
			}
			if len(storage.objectsSigned()) != 1 {
				t.Errorf("objek ditandatangani = %v, ingin tepat satu", storage.objectsSigned())
			}
		})
	}
}

// A returning customer's account spans several tours (v3.0), so the files of an
// earlier tour are still their own — the same ownership rule PortalTripInvoicePDF
// already applies to past invoices.
func TestPortalSignedURL_SignsFileFromEarlierTrip(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedTwoParticipants(h)

	h.asParticipant("participant-a", "portal-user-a").
		GET("/api/v1/portal/signed-url?type=document&id=document-participant-a-2019").
		expectCode(http.StatusOK)
}

// The endpoint no longer signs a location the caller supplies. Sending the old
// parameters must not reach storage — otherwise the old hole is still open for
// any client that keeps calling the old way.
func TestPortalSignedURL_RefusesRawBucketAndPath(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedTwoParticipants(h)

	h.asParticipant("participant-a", "portal-user-a").
		GET("/api/v1/portal/signed-url?bucket=participant-documents&path=participant-b/passport.jpg").
		expectCode(http.StatusBadRequest)

	storage.expectNothingSigned(t)
}

func TestPortalSignedURL_RefusesUnknownFileKind(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedTwoParticipants(h)

	h.asParticipant("participant-a", "portal-user-a").
		GET("/api/v1/portal/signed-url?type=invoice_pdf&id=invoice-participant-a").
		expectCode(http.StatusBadRequest)

	storage.expectNothingSigned(t)
}

// ─── Portal path: the claimed file path ───────────────────────────────────────

// Ownership is checked against the row, and a participant writes their own rows:
// both a document and a payment proof carry a file path the client supplies. So
// the path is untrusted input, and without this the ownership check above is
// satisfiable in two steps — record another participant's object as your own
// document, then ask for a URL to your own document.
func TestPortalUpload_RefusesFilePathOutsideOwnStorage(t *testing.T) {
	foreign := []struct {
		name string
		path string
	}{
		{"milik peserta lain", "participant-b/passport.jpg"},
		{"lolos lewat penelusuran direktori", "participant-a/../participant-b/passport.jpg"},
		// Escaped, the traversal survives a literal-slash split and still reaches
		// storage as one: net/http keeps "%2f" on the wire.
		{"penelusuran direktori tersandi", "participant-a/..%2fparticipant-b/passport.jpg"},
	}

	for _, tc := range foreign {
		t.Run("dokumen/"+tc.name, func(t *testing.T) {
			h := newHarness(t)
			seedTwoParticipants(h)
			before := len(h.Documents.order)

			h.asParticipant("participant-a", "portal-user-a").
				POST("/api/v1/portal/documents", map[string]string{
					"document_type": "passport",
					"file_path":     tc.path,
					"file_name":     "passport.jpg",
				}).expectCode(http.StatusBadRequest)

			if len(h.Documents.order) != before {
				t.Error("baris dokumen tetap dibuat meski path ditolak")
			}
		})

		t.Run("bukti bayar/"+tc.name, func(t *testing.T) {
			h := newHarness(t)
			seedTwoParticipants(h)
			before := len(h.Proofs.proofs)

			h.asParticipant("participant-a", "portal-user-a").
				POST("/api/v1/portal/invoices/invoice-participant-a/proofs", map[string]any{
					"file_path":      tc.path,
					"amount_claimed": 5000000,
				}).expectCode(http.StatusBadRequest)

			if len(h.Proofs.proofs) != before {
				t.Error("baris bukti bayar tetap dibuat meski path ditolak")
			}
		})
	}
}

// The check is about whose storage prefix the path names, not about rejecting
// uploads: a path produced by the participant's own upload goes through.
func TestPortalUpload_AcceptsOwnStoragePath(t *testing.T) {
	h := newHarness(t)
	seedTwoParticipants(h)

	h.asParticipant("participant-a", "portal-user-a").
		POST("/api/v1/portal/documents", map[string]string{
			"document_type": "passport",
			"file_path":     "participant-a/1754000000-passport.jpg",
			"file_name":     "passport.jpg",
		}).expectCode(http.StatusCreated)

	h.asParticipant("participant-a", "portal-user-a").
		POST("/api/v1/portal/invoices/invoice-participant-a/proofs", map[string]any{
			"file_path":      "participant-a/1754000000-transfer.jpg",
			"amount_claimed": 5000000,
		}).expectCode(http.StatusCreated)
}

// ─── Staff path: role ─────────────────────────────────────────────────────────

// §5.3 does not cover this endpoint, so its access level is a decision, recorded
// in the ticket: it sits on the ops group, because the files it unlocks are the
// ones listed by the ops-only document and invoice routes. A konsultan or tour
// leader who cannot list a participant's documents must not be able to sign one.
func TestStaffSignedURL_OnlyOpsRolesMaySign(t *testing.T) {
	opsRoles := map[string]bool{"super_admin": true, "admin": true}

	for _, file := range privateFiles {
		for _, role := range staffRoles {
			t.Run(file.name+"/"+role, func(t *testing.T) {
				storage := newSigningStorage(t)
				h := newHarness(t, withStorageServer(storage.URL))
				seedTwoParticipants(h)

				res := h.as(role).GET("/api/v1/admin/signed-url?" + file.query("participant-b"))
				if opsRoles[role] {
					res.expectCode(http.StatusOK)
					return
				}
				res.expectCode(http.StatusForbidden)
				storage.expectNothingSigned(t)
			})
		}
	}
}

func TestStaffSignedURL_RefusesRawBucketAndPath(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedTwoParticipants(h)

	h.as("admin").
		GET("/api/v1/admin/signed-url?bucket=participant-documents&path=participant-b/passport.jpg").
		expectCode(http.StatusBadRequest)

	storage.expectNothingSigned(t)
}

// A resource id that resolves to nothing is a 404, and never a signature over a
// path assembled from the id.
func TestStaffSignedURL_RefusesUnknownResource(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedTwoParticipants(h)

	h.as("admin").GET("/api/v1/admin/signed-url?type=document&id=tidak-ada").
		expectCode(http.StatusNotFound)

	storage.expectNothingSigned(t)
}

// ─── Token separation (§19.1), both directions ────────────────────────────────

// A staff token is signed with the same secret as a portal token, so only the
// claims tell them apart. The staff middleware already refuses a portal token;
// this is the missing half.
func TestPortalRoutes_RejectStaffToken(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	staffAsPortal := h.anonymous().withHeader("X-Portal-Token", h.staffToken("user-admin", "admin"))
	staffAsPortal.GET("/api/v1/portal/me").expectCode(http.StatusUnauthorized)
}

// A token with no participant id identifies nobody. Accepting it would run every
// portal handler with an empty participant id, which reads as "no rows" rather
// than as the refusal it should be.
func TestPortalRoutes_RejectTokenWithoutParticipant(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	anonymousPortal := h.anonymous().withHeader("X-Portal-Token", h.portalToken("", ""))
	anonymousPortal.GET("/api/v1/portal/me").expectCode(http.StatusUnauthorized)
}

// ─── The last super admin ─────────────────────────────────────────────────────

// Only super_admin may manage users (§5.3), so an organisation with no active
// super admin can never create one — including one that could undo the change
// that emptied the role.
func TestUserManagement_RefusesToDemoteLastSuperAdmin(t *testing.T) {
	h := newHarness(t)

	res := h.as("super_admin").PUT("/api/v1/admin/users/user-super_admin",
		map[string]string{"role": "admin"})
	res.expectCode(http.StatusConflict)
	if msg := res.message(); !strings.Contains(strings.ToLower(msg), "super admin") {
		t.Errorf("pesan %q tidak menjelaskan alasan penolakan", msg)
	}

	u, err := h.Users.GetByID(context.Background(), "user-super_admin")
	if err != nil {
		t.Fatalf("baca pengguna: %v", err)
	}
	if u.Role != "super_admin" {
		t.Errorf("peran = %q setelah penolakan, ingin tetap super_admin", u.Role)
	}
}

func TestUserManagement_RefusesToDeactivateLastSuperAdmin(t *testing.T) {
	h := newHarness(t)

	res := h.as("super_admin").PATCH("/api/v1/admin/users/user-super_admin/deactivate", nil)
	res.expectCode(http.StatusConflict)

	u, err := h.Users.GetByID(context.Background(), "user-super_admin")
	if err != nil {
		t.Fatalf("baca pengguna: %v", err)
	}
	if !u.IsActive {
		t.Error("akun dinonaktifkan meski permintaan ditolak")
	}
}

// The guard is about the last one, not about super admins in general: with a
// second active super admin the same operations go through.
func TestUserManagement_AllowsDemotingOneOfTwoSuperAdmins(t *testing.T) {
	h := newHarness(t)
	seedSecondSuperAdmin(h)

	h.as("super_admin").PUT("/api/v1/admin/users/user-super_admin",
		map[string]string{"role": "admin"}).expectCode(http.StatusOK)
}

func TestUserManagement_AllowsDeactivatingOneOfTwoSuperAdmins(t *testing.T) {
	h := newHarness(t)
	seedSecondSuperAdmin(h)

	h.as("super_admin").PATCH("/api/v1/admin/users/user-super_admin-2/deactivate", nil).
		expectCode(http.StatusOK)
}

func seedSecondSuperAdmin(h *harness) {
	h.Users.Seed(domainUser.User{
		ID: "user-super_admin-2", Name: "Super Dua", Email: "super2@pintour.test",
		Role: "super_admin", IsActive: true,
	})
}

// An inactive super admin cannot log in, so it is not one of the accounts that
// keeps the organisation reachable.
func TestUserManagement_IgnoresInactiveSuperAdminsWhenCounting(t *testing.T) {
	h := newHarness(t)
	h.Users.Seed(domainUser.User{
		ID: "user-super_admin-lama", Name: "Super Lama", Email: "lama@pintour.test",
		Role: "super_admin", IsActive: false,
	})

	h.as("super_admin").PATCH("/api/v1/admin/users/user-super_admin/deactivate", nil).
		expectCode(http.StatusConflict)
}
