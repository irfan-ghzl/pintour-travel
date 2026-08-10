package httpdelivery

// Lead status audit trail (tiket 06).
//
// FR-CRM-02: "Setiap perubahan status harus tercatat dengan timestamp dan
// pengguna yang melakukan perubahan". The repository used to take the actor and
// throw it away — the parameter was named `_` — and there was no history table
// at all, so the question "who moved this lead to deal?" had no answer anywhere
// in the system.

import (
	"net/http"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
)

func seedLeadForAudit(h *harness) {
	h.Leads.Seed(domainLead.Lead{
		ID: "lead-1", Name: "Prospek Satu", Phone: "628770000001",
		PackageID: "package-1", Status: "baru", CreatedAt: time.Now(),
	})
}

// statusHistory reads the audit trail back through the admin lead detail, which
// is where FR-CRM-02 has to be visible — not from the fake directly.
func statusHistory(t *testing.T, h *harness, leadID string) []map[string]any {
	t.Helper()
	res := h.as("admin").GET("/api/v1/admin/leads/" + leadID)
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data struct {
			StatusHistory []map[string]any `json:"status_history"`
		} `json:"data"`
	}
	res.decode(&envelope)
	return envelope.Data.StatusHistory
}

// The criterion this ticket names: two people move the same lead, and the trail
// says which of them did what. Before, both transitions were invisible.
func TestLeadStatus_RecordsWhoChangedIt(t *testing.T) {
	h := newHarness(t)
	seedLeadForAudit(h)

	h.asUser("user-konsultan", "konsultan").
		PATCH("/api/v1/admin/leads/lead-1/status", map[string]string{"status": "dihubungi"}).
		expectCode(http.StatusOK)
	h.asUser("user-admin", "admin").
		PATCH("/api/v1/admin/leads/lead-1/status", map[string]string{"status": "deal"}).
		expectCode(http.StatusOK)

	history := statusHistory(t, h, "lead-1")
	if len(history) != 2 {
		t.Fatalf("baris riwayat = %d, ingin 2: %v", len(history), history)
	}

	first, second := history[0], history[1]
	if first["to_status"] != "dihubungi" || second["to_status"] != "deal" {
		t.Errorf("urutan riwayat salah: %v → %v", first["to_status"], second["to_status"])
	}
	if first["from_status"] != "baru" || second["from_status"] != "dihubungi" {
		t.Errorf("status lama tidak tercatat: %v, %v", first["from_status"], second["from_status"])
	}
	if first["changed_by"] == second["changed_by"] {
		t.Errorf("kedua perubahan tercatat atas pelaku yang sama (%v)", first["changed_by"])
	}
	if first["changed_by"] != "user-konsultan" || second["changed_by"] != "user-admin" {
		t.Errorf("pelaku salah: %v, %v", first["changed_by"], second["changed_by"])
	}
}

// The trail has to sit next to the consultant's own notes, because reading one
// without the other loses the thread of what happened to the lead.
func TestLeadDetail_ShowsStatusHistoryBesideNotes(t *testing.T) {
	h := newHarness(t)
	seedLeadForAudit(h)

	h.as("konsultan").POST("/api/v1/admin/leads/lead-1/notes",
		map[string]string{"note": "Sudah ditelepon, minta ditawarkan paket lain"}).
		expectCode(http.StatusCreated)
	h.as("konsultan").PATCH("/api/v1/admin/leads/lead-1/status",
		map[string]string{"status": "konsultasi"}).expectCode(http.StatusOK)

	res := h.as("admin").GET("/api/v1/admin/leads/lead-1")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data struct {
			ActivityLog   []map[string]any `json:"activity_log"`
			StatusHistory []map[string]any `json:"status_history"`
		} `json:"data"`
	}
	res.decode(&envelope)

	if len(envelope.Data.StatusHistory) != 1 {
		t.Errorf("riwayat status = %d, ingin 1", len(envelope.Data.StatusHistory))
	}
	// One note, the consultant's own. The status change used to be written into
	// this list as a synthetic "[SISTEM]" note; now it has a place of its own,
	// so the consultant's notes stay theirs.
	if n := len(envelope.Data.ActivityLog); n != 1 {
		t.Errorf("catatan konsultan = %d, ingin 1 (tanpa catatan sintetis): %v",
			n, envelope.Data.ActivityLog)
	}
}

// Converting a lead is a status change like any other, and the trail says who
// converted it.
func TestConvertLead_RecordedInStatusHistory(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusCreated)

	history := statusHistory(t, h, "lead-1")
	if len(history) != 1 {
		t.Fatalf("baris riwayat = %d, ingin 1: %v", len(history), history)
	}
	if history[0]["to_status"] != "peserta" || history[0]["from_status"] != "deal" {
		t.Errorf("konversi tercatat sebagai %v → %v, ingin deal → peserta",
			history[0]["from_status"], history[0]["to_status"])
	}
	if history[0]["changed_by"] != "user-admin" {
		t.Errorf("pelaku konversi = %v, ingin user-admin", history[0]["changed_by"])
	}
}

// A rolled-back conversion leaves no history row either: the trail records what
// happened, and that conversion did not.
func TestConvertLead_RolledBackLeavesNoStatusHistory(t *testing.T) {
	h := newHarness(t)
	seedConvertible(h)
	h.Participants.Fail(errWriteFailed)

	h.as("admin").POST("/api/v1/admin/participants/convert", convertRequest("double")).
		expectCode(http.StatusUnprocessableEntity)

	if history := statusHistory(t, h, "lead-1"); len(history) != 0 {
		t.Errorf("riwayat = %d baris setelah konversi di-rollback, ingin 0: %v",
			len(history), history)
	}
}

// ─── Soft delete ──────────────────────────────────────────────────────────────

// The invoice repository was the only one that never filtered its soft-delete
// column, so a deleted invoice still appeared in listings — a ghost bill.
func TestInvoices_SoftDeletedRowsAreHidden(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	deleted := time.Now()
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-dihapus", InvoiceNumber: "INV-202608-0009",
		ParticipantID: "participant-1", BatchID: "batch-1",
		Amount: 1000000, Status: "diterbitkan",
		DueDate: calendar.Today().AddDays(1), DeletedAt: &deleted,
	})

	res := h.as("admin").GET("/api/v1/admin/invoices")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res.decode(&envelope)
	for _, inv := range envelope.Data {
		if inv.ID == "invoice-dihapus" {
			t.Error("invoice terhapus lunak masih muncul di daftar")
		}
	}

	h.as("admin").GET("/api/v1/admin/invoices/invoice-dihapus").
		expectCode(http.StatusNotFound)
}

// Numbering must not walk back over a deleted row's number: two invoices issued
// either side of a deletion have to stay distinct (§13.7). The hiding above and
// this are in tension — a numbering scheme that counts only visible rows would
// hand the deleted one's number straight back.
func TestInvoices_NumberingSkipsSoftDeletedRows(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	issue := func() (id, number string) {
		res := h.as("admin").POST("/api/v1/admin/invoices", map[string]any{
			"participant_id": "participant-1",
			"batch_id":       "batch-1",
			"amount":         5000000,
			"due_date":       calendar.Today().AddDays(7).Format(time.RFC3339),
		})
		res.expectCode(http.StatusCreated)
		data := res.data()
		id, _ = data["id"].(string)
		number, _ = data["invoice_number"].(string)
		return id, number
	}

	firstID, firstNumber := issue()
	h.Invoices.SoftDelete(firstID)
	_, secondNumber := issue()

	if firstNumber == "" || firstNumber == secondNumber {
		t.Errorf("nomor invoice dipakai ulang setelah penghapusan lunak: %q lalu %q",
			firstNumber, secondNumber)
	}
}
