package httpdelivery

// Lead conversion (tiket 05).
//
// Converting a lead writes three rows that only make sense together: the portal
// identity the customer logs in with, the participant row that identity points
// at, and the lead marked as converted. These tests are written as "what is left
// behind when a step fails", because the defect they exist for left no error
// anywhere — the conversion reported failure, an orphan portal identity stayed,
// and the retry then recognised that orphan as a returning customer and issued
// no password. The participant was locked out permanently, and portal accounts
// have no password-reset flow.

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
)

// convertPhone is the lead's phone throughout: the portal identity is keyed on
// it, so it is what "was an account left behind?" is asked about.
const convertPhone = "628990000001"

// seedConvertible puts a lead in the one state conversion accepts, with the
// batch and package the convert-time automation reads.
func seedConvertible(h *harness) {
	h.Packages.Seed(domainPkg.Package{
		ID: "package-1", Name: "Umroh Reguler 9 Hari", Slug: "umroh-reguler-9-hari",
		Destination: "Arab Saudi", Category: "umroh", DurationDays: 9,
		BasePrice: 25000000, IsActive: true,
	})
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-1", PackageID: "package-1",
		DepartureDate: calendar.Today().AddDays(60),
		ReturnDate:    calendar.Today().AddDays(69),
		PriceDouble:   25000000, PriceSingle: 30000000, PriceTriple: 22000000,
		Quota: 40, Status: "tersedia",
	})
	h.Leads.Seed(domainLead.Lead{
		ID: "lead-1", Name: "Calon Peserta", Phone: convertPhone,
		Email: "calon@pintour.test", PackageID: "package-1", Pax: 2,
		Status: "deal", CreatedAt: time.Now(),
	})
}

// convertRequest is the body the admin form posts.
func convertRequest(roomType string) map[string]string {
	return map[string]string{"lead_id": "lead-1", "batch_id": "batch-1", "room_type": roomType}
}

// portalUserFor returns the portal identity stored for a phone, or nil.
func portalUserFor(h *harness, phone string) *portaluser.PortalUser {
	pu, err := h.PortalUsers.GetByPhone(context.Background(), phone)
	if err != nil {
		return nil
	}
	return pu
}

// ─── Nothing left behind ──────────────────────────────────────────────────────

// The defect this ticket exists for. The portal identity was written first, in
// its own statement; when the participant write failed the identity survived,
// holding a password hash that had reached nobody.
func TestConvertLead_FailedParticipantWriteLeavesNoPortalIdentity(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)
	h.Participants.Fail(errWriteFailed)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusUnprocessableEntity)

	if pu := portalUserFor(h, convertPhone); pu != nil {
		t.Errorf("identitas portal %q tertinggal setelah konversi gagal", pu.ID)
	}
	if h.Unit.RolledBack != 1 || h.Unit.Committed != 0 {
		t.Errorf("unit: %d commit / %d rollback, ingin 0 dan 1",
			h.Unit.Committed, h.Unit.RolledBack)
	}
	lead, err := h.Leads.GetByID(context.Background(), "lead-1")
	if err != nil {
		t.Fatalf("baca lead: %v", err)
	}
	if lead.Status != "deal" || lead.ConvertedAt != nil {
		t.Errorf("lead = %q (converted_at=%v), ingin tetap deal dan belum terkonversi",
			lead.Status, lead.ConvertedAt)
	}
}

// The unit is all three rows, not just the first two: a failure on the last step
// unwinds the participant and the portal identity with it.
func TestConvertLead_FailedLeadMarkLeavesNoParticipant(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)
	h.Leads.FailMarkConverted(errWriteFailed)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusUnprocessableEntity)

	if n := len(h.Participants.order); n != 0 {
		t.Errorf("peserta tertulis = %d, ingin 0", n)
	}
	if pu := portalUserFor(h, convertPhone); pu != nil {
		t.Error("identitas portal tertinggal setelah penandaan lead gagal")
	}
}

// The first write in the unit failing is the case with nothing yet to undo — it
// still must not produce a participant, and the lead must stay convertible.
func TestConvertLead_FailedPortalIdentityWriteLeavesNothing(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)
	h.PortalUsers.Fail(errWriteFailed)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusUnprocessableEntity)

	if n := len(h.Participants.order); n != 0 {
		t.Errorf("peserta tertulis = %d, ingin 0", n)
	}
	lead, _ := h.Leads.GetByID(context.Background(), "lead-1")
	if lead.Status != "deal" {
		t.Errorf("lead = %q, ingin tetap deal", lead.Status)
	}
}

// One lead becomes at most one participant. The status check runs before the
// transaction opens, so two requests can both pass it; the second is stopped by
// the database, and the participant it had already written goes back with it.
func TestConvertLead_SecondConversionOfSameLeadIsRefused(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusCreated)
	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusUnprocessableEntity)

	if n := len(h.Participants.order); n != 1 {
		t.Errorf("peserta untuk satu lead = %d, ingin 1", n)
	}
	if n := len(h.PortalUsers.order); n != 1 {
		t.Errorf("identitas portal = %d, ingin 1", n)
	}
}

// The consequence that made the original defect permanent: after a failed
// attempt the customer must still look new, so the retry issues a password.
func TestConvertLead_RetryAfterFailureIssuesNewPassword(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)

	h.Participants.Fail(errWriteFailed)
	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusUnprocessableEntity)

	h.Participants.Fail(nil)
	res := h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double"))
	res.expectCode(http.StatusCreated)

	data := res.data()
	if reused, _ := data["reused_account"].(bool); reused {
		t.Error("percobaan ulang mengenali peserta sebagai pelanggan lama")
	}
	if pw, _ := data["temp_password"].(string); pw == "" {
		t.Error("percobaan ulang tidak menerbitkan password baru")
	}
}

// ─── The successful unit ──────────────────────────────────────────────────────

func TestConvertLead_WritesAllThreeRows(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)

	res := h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("single"))
	res.expectCode(http.StatusCreated)

	pu := portalUserFor(h, convertPhone)
	if pu == nil {
		t.Fatal("identitas portal tidak dibuat")
	}
	if len(h.Participants.order) != 1 {
		t.Fatalf("peserta tertulis = %d, ingin 1", len(h.Participants.order))
	}
	p := h.Participants.participants[h.Participants.order[0]]
	if p.PortalUserID == nil || *p.PortalUserID != pu.ID {
		t.Errorf("peserta tidak tertaut ke identitas portal (%v)", p.PortalUserID)
	}
	lead, _ := h.Leads.GetByID(context.Background(), "lead-1")
	if lead.Status != "peserta" || lead.ConvertedAt == nil {
		t.Errorf("lead = %q (converted_at=%v), ingin peserta dan terkonversi",
			lead.Status, lead.ConvertedAt)
	}
}

// FR-CRM-08: one account across trips. A returning customer keeps the password
// they already have, so no new one is issued and no second identity is created.
func TestConvertLead_ReturningCustomerReusesAccount(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)
	h.PortalUsers.Seed(portaluser.PortalUser{
		ID: "portal-user-lama", Phone: convertPhone, Name: "Calon Peserta",
		PasswordHash: "$2a$12$hashlama",
	})

	res := h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double"))
	res.expectCode(http.StatusCreated)

	data := res.data()
	if reused, _ := data["reused_account"].(bool); !reused {
		t.Error("pelanggan lama tidak dikenali")
	}
	if pw, _ := data["temp_password"].(string); pw != "" {
		t.Error("password baru diterbitkan untuk pelanggan lama")
	}
	if n := len(h.PortalUsers.order); n != 1 {
		t.Errorf("identitas portal = %d, ingin tetap 1", n)
	}
	p := h.Participants.participants[h.Participants.order[0]]
	if p.PortalUserID == nil || *p.PortalUserID != "portal-user-lama" {
		t.Errorf("peserta tidak tertaut ke akun lama (%v)", p.PortalUserID)
	}
}

// ─── Room type ────────────────────────────────────────────────────────────────

// A room type the schema does not know used to fail partway through the writes.
// It is refused before any of them.
func TestConvertLead_RejectsUnknownRoomTypeBeforeWriting(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("presidential")).
		expectCode(http.StatusBadRequest)

	if len(h.Participants.order) != 0 || len(h.PortalUsers.order) != 0 {
		t.Error("konversi menulis baris meski tipe kamar ditolak")
	}
	// Refused before anything was attempted, not rolled back afterwards: the unit
	// was never entered at all.
	if h.Unit.Committed != 0 || h.Unit.RolledBack != 0 {
		t.Errorf("unit dijalankan (%d commit / %d rollback) meski tipe kamar tidak dikenal",
			h.Unit.Committed, h.Unit.RolledBack)
	}
	lead, _ := h.Leads.GetByID(context.Background(), "lead-1")
	if lead.Status != "deal" {
		t.Errorf("lead = %q, ingin tetap deal", lead.Status)
	}
}

// ─── What the admin is told ───────────────────────────────────────────────────

// The old message told the admin to send the password and nothing ever did.
// FR-PORTAL-01 has the system send it — but only where there is a gateway to
// send through, and this harness has none. So what is asserted here is that the
// admin is told which of those two worlds they are in: with no gateway the
// response reports the password unsent and asks for it to be relayed, rather
// than assuring the admin of a delivery that never happened.
//
// The other branch — gateway configured, message claims automatic delivery —
// cannot be reached from here: enabling the gateway would put these tests on the
// network. It becomes assertable once the Fonnte base URL is settable, which is
// the first criterion of tiket 08.
func TestConvertLead_TellsAdminTheTruthAboutTheCredentialSend(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)

	res := h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double"))
	res.expectCode(http.StatusCreated)
	data := res.data()

	if sent, _ := data["credentials_sent"].(bool); sent {
		t.Error("respons mengklaim password terkirim padahal tidak ada gateway WA")
	}
	message, _ := data["message"].(string)
	if !strings.Contains(strings.ToLower(message), "kirimkan password") {
		t.Errorf("pesan %q tidak meminta admin meneruskan password, padahal sistem tidak mengirimnya", message)
	}
	// The password still comes back either way: it is the only copy that exists.
	if pw, _ := data["temp_password"].(string); pw == "" {
		t.Error("password sementara tidak dikembalikan ke admin")
	}
}
