package httpdelivery

// Participant portal completeness (tiket 09).
//
// These are features that stopped short of what the documents promise rather
// than behaviour that goes wrong, so the tests read as "the portal offers this
// now" — except the last, which guards the thing that must not come with them:
// one portal identity reaching another's archive.

import (
	"encoding/json"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
	"net/http"
	"testing"
)

// seedTwoPortalIdentities gives two unrelated customers one past trip each, so
// "another portal identity's archive" is a real thing to ask for.
//
// Both trips are settled (is_active), which is what "riwayat perjalanan lampau,
// yang memang sudah dibayar" means: the archive is open because those trips were
// paid for, not because their dates have passed. A test that needs an unpaid one
// says so over the top — TestMyTrips_UnpaidPastTripReadsAsCancelled does.
func seedTwoPortalIdentities(h *harness) {
	h.Packages.Seed(domainPkg.Package{
		ID: "package-1", Name: "Umroh Reguler 9 Hari", Slug: "umroh-reguler",
		Destination: "Arab Saudi", DurationDays: 9, Itinerary: json.RawMessage(`["Hari 1: tiba di Jeddah"]`), IsActive: true,
	})
	lastYear := calendar.Today().AddDays(-365)
	lastYearAt := lastYear.Time()
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-lama", PackageID: "package-1",
		DepartureDate: lastYear, ReturnDate: lastYear.AddDays(9),
		Quota: 40, PriceDouble: 25000000, Status: "tersedia",
	})

	for _, c := range []struct{ portalUser, participant, phone string }{
		{"portal-user-a", "participant-a", "628111000001"},
		{"portal-user-b", "participant-b", "628111000002"},
	} {
		h.PortalUsers.Seed(portaluser.PortalUser{ID: c.portalUser, Phone: c.phone, Name: c.participant})
		portalUser := c.portalUser
		h.Participants.Seed(domainParticipant.Participant{
			ID: c.participant, PortalUserID: &portalUser, BatchID: "batch-lama",
			Name: c.participant, Phone: c.phone, RoomType: "double",
			Email: c.participant + "@pintour.test", IsActive: true,
			BatchDepartureDate: &lastYearAt, PackageName: "Umroh Reguler 9 Hari",
		})
	}
}

// ─── FR-PORTAL-09: the completion badge ───────────────────────────────────────

// A past trip that was paid in full happened; one that never was, did not. The
// schema has no cancelled flag, so the badge is derived — see the ticket for why
// that was chosen over adding a column nothing would set.
func TestMyTrips_PastTripsCarryACompletionBadge(t *testing.T) {
	h := newHarness(t)
	seedTwoPortalIdentities(h)
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-a", InvoiceNumber: "INV-202508-0001", ParticipantID: "participant-a",
		BatchID: "batch-lama", Amount: 25000000, Status: "lunas",
		DueDate: calendar.Today().AddDays(-300),
	})

	res := h.asParticipant("participant-a", "portal-user-a").GET("/api/v1/portal/my-trips")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data struct {
			History []map[string]any `json:"history"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if len(envelope.Data.History) != 1 {
		t.Fatalf("riwayat = %d kartu, ingin 1: %v", len(envelope.Data.History), envelope.Data.History)
	}
	if got := envelope.Data.History[0]["completion_status"]; got != "selesai" {
		t.Errorf("badge = %v, ingin selesai (perjalanan lunas dan sudah lewat)", got)
	}
}

func TestMyTrips_UnpaidPastTripReadsAsCancelled(t *testing.T) {
	h := newHarness(t)
	seedTwoPortalIdentities(h)
	// The one trip in this file that was never paid for: a departure that came
	// and went with no invoice ever settled.
	unpaid := *h.Participants.participants["participant-a"]
	unpaid.IsActive = false
	h.Participants.Seed(unpaid)

	res := h.asParticipant("participant-a", "portal-user-a").GET("/api/v1/portal/my-trips")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data struct {
			History []map[string]any `json:"history"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if len(envelope.Data.History) != 1 {
		t.Fatalf("riwayat = %d kartu, ingin 1", len(envelope.Data.History))
	}
	if got := envelope.Data.History[0]["completion_status"]; got != "dibatalkan" {
		t.Errorf("badge = %v, ingin dibatalkan (tidak pernah lunas)", got)
	}
}

// ─── FR-PORTAL-10: the archive ────────────────────────────────────────────────

func TestTripItinerary_OwnPastTripIsDownloadable(t *testing.T) {
	h := newHarness(t)
	seedTwoPortalIdentities(h)

	res := h.asParticipant("participant-a", "portal-user-a").
		GET("/api/v1/portal/my-trips/participant-a/itinerary")
	res.expectCode(http.StatusOK)

	if name, _ := res.data()["package_name"].(string); name != "Umroh Reguler 9 Hari" {
		t.Errorf("itinerary tidak memuat nama paket: %s", res.Body)
	}
}

// The guard that has to come with the archive. A 404 rather than a 403, for the
// same reason as the signed-URL endpoint: 403 would confirm the trip exists.
func TestTripArtefacts_RefuseAnotherPortalIdentity(t *testing.T) {
	for _, path := range []string{
		"/api/v1/portal/my-trips/participant-b/itinerary",
		"/api/v1/portal/my-trips/participant-b/invoice-pdf",
	} {
		t.Run(path, func(t *testing.T) {
			h := newHarness(t)
			seedTwoPortalIdentities(h)
			h.Invoices.Seed(domainInvoice.Invoice{
				ID: "invoice-b", InvoiceNumber: "INV-202508-0002", ParticipantID: "participant-b",
				BatchID: "batch-lama", Amount: 25000000, Status: "lunas",
				DueDate: calendar.Today().AddDays(-300),
			})

			h.asParticipant("participant-a", "portal-user-a").GET(path).
				expectCode(http.StatusNotFound)
		})
	}
}

// ─── FR-PORTAL-12: the consultation form ──────────────────────────────────────

// The public form is prefilled from the server, not from whatever the browser
// remembered — a name an admin corrected last week has to arrive corrected.
func TestConsultationPrefill_ReturnsTheParticipantsOwnDetails(t *testing.T) {
	h := newHarness(t)
	seedTwoPortalIdentities(h)

	res := h.asParticipant("participant-a", "portal-user-a").
		GET("/api/v1/portal/consultation-prefill")
	res.expectCode(http.StatusOK)

	data := res.data()
	for field, want := range map[string]string{
		"name":      "participant-a",
		"phone":     "628111000001",
		"email":     "participant-a@pintour.test",
		"room_type": "double",
	} {
		if got, _ := data[field].(string); got != want {
			t.Errorf("%s = %q, ingin %q", field, got, want)
		}
	}
}

func TestConsultationPrefill_RequiresAPortalToken(t *testing.T) {
	h := newHarness(t)
	seedTwoPortalIdentities(h)

	h.anonymous().GET("/api/v1/portal/consultation-prefill").
		expectCode(http.StatusUnauthorized)
}

// The public lead endpoint decides returning-customer status from the phone
// number itself, so a client claiming it is ignored (FR-CRM-07 / §19).
func TestPublicLead_IgnoresClientClaimedReturningStatus(t *testing.T) {
	h := newHarness(t)
	seedTwoPortalIdentities(h)

	res := h.anonymous().POST("/api/v1/leads", map[string]any{
		"name":           "Orang Lain",
		"phone":          "628999999999", // no portal account
		"package_id":     "package-1",
		"pax":            2,
		"is_returning":   true,
		"portal_user_id": "portal-user-a",
	})
	res.expectCode(http.StatusCreated)

	id, _ := res.data()["id"].(string)
	lead, err := h.Leads.GetByID(t.Context(), id)
	if err != nil {
		t.Fatalf("baca lead: %v", err)
	}
	if lead.IsReturning || lead.PortalUserID != nil {
		t.Errorf("status pelanggan lama dipercaya dari klien: is_returning=%v portal_user_id=%v",
			lead.IsReturning, lead.PortalUserID)
	}
}

// ─── FR-INV-03: the stage that was never reached ──────────────────────────────

// diterbitkan → menunggu_bayar → dibayar → lunas. An approved partial payment
// used to leave the invoice on menunggu_bayar, indistinguishable from one nobody
// had paid anything towards.
func TestInvoiceStatus_PartialPaymentReachesDibayar(t *testing.T) {
	h := newHarness(t)
	seedFullyPayable(h)
	h.Proofs.Seed(domainInvoice.PaymentProof{
		ID: "proof-1", InvoiceID: "invoice-1", FilePath: "participant-1/transfer.jpg",
		AmountClaimed: 2500000, Status: "menunggu",
	})

	h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-1/review",
		map[string]string{"status": "disetujui", "notes": "uang muka"}).
		expectCode(http.StatusOK)

	inv, err := h.Invoices.GetByID(t.Context(), "invoice-1")
	if err != nil {
		t.Fatalf("baca invoice: %v", err)
	}
	if inv.Status != "dibayar" {
		t.Errorf("status = %q setelah pembayaran sebagian disetujui, ingin dibayar", inv.Status)
	}
}
