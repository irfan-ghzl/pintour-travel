package httpdelivery

// Two responses that carried the wrong status (tiket 17).
//
// Both were found by driving the running API rather than by reading it, and both
// look the same from the outside: a 500 where nothing was actually broken. A 500
// costs twice — the caller is told to retry something that will never succeed,
// and the log fills with failures that are not failures, hiding the ones that
// are.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
)

// ─── Private files recorded as absolute URLs ─────────────────────────────────

// seedManualURLFile gives one participant a document and a payment proof whose
// stored path is an absolute URL — what a deployment without storage keys writes
// (the manual-URL fallback), and what the seed data contains.
func seedManualURLFile(h *harness) {
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-manual", BatchID: "batch-1", Name: "Peserta Manual",
		Phone: "628777000111", IsActive: true,
	})
	h.PortalUsers.Seed(portaluser.PortalUser{ID: "portal-user-manual", Phone: "628777000111", Name: "Peserta Manual"})
	h.Documents.Seed(domainDocument.Document{
		ID: "document-manual", ParticipantID: "participant-manual",
		DocumentType: "passport", FilePath: "https://example.com/docs/Paspor.pdf",
		FileName: "Paspor.pdf", Status: "menunggu",
	})
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-manual", InvoiceNumber: "INV-MANUAL", ParticipantID: "participant-manual",
		BatchID: "batch-1", Amount: 1000000, Status: "menunggu_bayar",
	})
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-manual", InvoiceID: "invoice-manual",
		FilePath: "https://example.com/docs/Transfer.jpg", AmountClaimed: 1000000, Status: "menunggu",
	})
}

// The file is already reachable and has no bucket object behind it, so the
// answer is the address itself. Signing it built ".../sign/<bucket>/https://..."
// and failed — a 500 on the one path the caller was entitled to.
func TestSignedURL_ReturnsAnAbsolutePathAsItIs(t *testing.T) {
	cases := []struct {
		name, path, query string
		want              string
	}{
		{"portal dokumen", "/api/v1/portal/signed-url", "type=document&id=document-manual", "https://example.com/docs/Paspor.pdf"},
		{"portal bukti bayar", "/api/v1/portal/signed-url", "type=payment_proof&id=proof-manual", "https://example.com/docs/Transfer.jpg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storage := newSigningStorage(t)
			h := newHarness(t, withStorageServer(storage.URL))
			seedManualURLFile(h)

			res := h.asParticipant("participant-manual", "portal-user-manual").
				GET(tc.path + "?" + tc.query)
			res.expectCode(http.StatusOK)

			if got, _ := res.data()["url"].(string); got != tc.want {
				t.Errorf("url = %q, ingin %q", got, tc.want)
			}
			storage.expectNothingSigned(t)
		})
	}
}

func TestStaffSignedURL_ReturnsAnAbsolutePathAsItIs(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedManualURLFile(h)

	res := h.as("admin").GET("/api/v1/admin/signed-url?type=document&id=document-manual")
	res.expectCode(http.StatusOK)

	if got, _ := res.data()["url"].(string); got != "https://example.com/docs/Paspor.pdf" {
		t.Errorf("url = %q, ingin URL tersimpan apa adanya", got)
	}
	storage.expectNothingSigned(t)
}

// The shortcut must not become a way around the ownership rule: an absolute URL
// belonging to someone else is still refused, and still with the 404 that keeps
// its existence undisclosed.
func TestPortalSignedURL_StillRefusesAnAbsolutePathOfAnotherParticipant(t *testing.T) {
	storage := newSigningStorage(t)
	h := newHarness(t, withStorageServer(storage.URL))
	seedManualURLFile(h)
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-lain", BatchID: "batch-1", Name: "Peserta Lain",
		Phone: "628777000222", IsActive: true,
	})
	h.PortalUsers.Seed(portaluser.PortalUser{ID: "portal-user-lain", Phone: "628777000222", Name: "Peserta Lain"})

	h.asParticipant("participant-lain", "portal-user-lain").
		GET("/api/v1/portal/signed-url?type=document&id=document-manual").
		expectCode(http.StatusNotFound)

	storage.expectNothingSigned(t)
}

// It must not turn a storage-less deployment into one that silently serves
// nothing either: a real bucket path with no storage configured is still 503.
func TestSignedURL_BucketPathStillNeedsStorage(t *testing.T) {
	h := newHarness(t) // no storage server: adapter unconfigured
	seedManualURLFile(h)
	h.Documents.Seed(domainDocument.Document{
		ID: "document-bucket", ParticipantID: "participant-manual",
		DocumentType: "ktp", FilePath: "participant-manual/ktp.jpg",
		FileName: "ktp.jpg", Status: "menunggu",
	})

	h.asParticipant("participant-manual", "portal-user-manual").
		GET("/api/v1/portal/signed-url?type=document&id=document-bucket").
		expectCode(http.StatusServiceUnavailable)
}

// ─── A public form naming something that does not exist ──────────────────────

// fkViolation is what the driver hands back when a write names a row that is not
// there — the shape the mapping reads, rather than the message text.
func fkViolation(table, constraint string) *pgconn.PgError {
	return &pgconn.PgError{
		Code:           "23503",
		Message:        `insert or update on table "` + table + `" violates foreign key constraint "` + constraint + `"`,
		TableName:      table,
		ConstraintName: constraint,
	}
}

// The lead form is public and only rate-limited, so a package_id that does not
// exist is ordinary traffic. It used to answer 500, which tells the visitor to
// try again and the operator that the system is broken — neither true.
func TestCreateLead_UnknownPackageIsARequestError(t *testing.T) {
	h := newHarness(t)
	h.Leads.Fail(fkViolation("leads", "leads_package_id_fkey"))

	res := h.anonymous().POST("/api/v1/leads", map[string]any{
		"name":       "Calon Peserta",
		"phone":      "08123456789",
		"package_id": "11111111-1111-1111-1111-111111111111",
	})
	res.expectCode(http.StatusBadRequest)

	if msg := res.message(); !strings.Contains(msg, "package_id") {
		t.Errorf("pesan = %q, ingin menyebut field package_id", msg)
	}
}

// A database that is genuinely failing must keep its 500: the mapping widens
// what counts as the caller's fault, and nothing else.
func TestCreateLead_RealDatabaseFailureStaysAServerError(t *testing.T) {
	h := newHarness(t)
	h.Leads.Fail(&pgconn.PgError{Code: "53300", Message: "too many connections"})

	h.anonymous().POST("/api/v1/leads", map[string]any{
		"name":       "Calon Peserta",
		"phone":      "08123456789",
		"package_id": "11111111-1111-1111-1111-111111111111",
	}).expectCode(http.StatusInternalServerError)
}
