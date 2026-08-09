package httpdelivery

// §25.5 Right to Erasure.
//
// The endpoint that receives these requests used to answer "diterima, akan
// diproses dalam 14 hari kerja sesuai UU PDP Pasal 46" and store nothing. The
// participant was given a legal commitment; nobody in the business ever learned
// they had asked. These tests are written around that: what the participant is
// promised has to be findable by someone who can act on it.

import (
	"context"
	"net/http"
	"testing"

	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/privacy"
)

func seedErasureCandidate(h *harness) {
	h.seedBaseline()
	h.Documents.Seed(domainDocument.Document{
		ID: "document-erasure-1", ParticipantID: "participant-1", DocumentType: "passport",
		FilePath: "participant-1/paspor.jpg", FileName: "paspor.jpg", Status: "disetujui",
	})
	h.Documents.Seed(domainDocument.Document{
		ID: "document-erasure-2", ParticipantID: "participant-1", DocumentType: "ktp",
		FilePath: "participant-1/ktp.jpg", FileName: "ktp.jpg", Status: "menunggu",
	})
}

// The promise and the record have to arrive together.
func TestPortalDeletionRequest_IsRecordedNotJustAcknowledged(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	res := h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request",
		map[string]any{"reason": "Tidak jadi berangkat"})
	res.expectCode(http.StatusOK)

	stored, err := h.Deletions.List(context.Background(), "")
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("%d requests recorded, want 1 — the participant was promised something nobody can act on", len(stored))
	}
	if stored[0].ParticipantID != "participant-1" || stored[0].Reason != "Tidak jadi berangkat" {
		t.Errorf("recorded %+v, want the requesting participant and their reason", stored[0])
	}
	if stored[0].Status != privacy.StatusPending {
		t.Errorf("status = %q, want %q", stored[0].Status, privacy.StatusPending)
	}
	// The ticket handed back has to name the stored request, or the participant
	// cannot ask about it later.
	if ticket, _ := res.data()["ticket"].(string); ticket == "DEL-" || ticket == "" {
		t.Errorf("ticket = %q, want one carrying the request id", ticket)
	}
}

// Pressing the button again is what someone does when they are not sure the
// first press worked. It must not put them in the queue twice.
func TestPortalDeletionRequest_AskingTwiceKeepsOneRequest(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	for i := 0; i < 3; i++ {
		h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request",
			map[string]any{"reason": "Tidak jadi berangkat"}).expectCode(http.StatusOK)
	}
	stored, _ := h.Deletions.List(context.Background(), "")
	if len(stored) != 1 {
		t.Errorf("%d requests after asking three times, want 1", len(stored))
	}
}

// The PRD writes this endpoint as DELETE; the portal client sends POST. Both
// have to work, or one of the two documents is lying.
func TestPortalDeletionRequest_AcceptsBothVerbs(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	h.asParticipant("participant-1").
		DELETE("/api/v1/portal/account-deletion-request").
		expectCode(http.StatusOK)

	if stored, _ := h.Deletions.List(context.Background(), ""); len(stored) != 1 {
		t.Errorf("DELETE recorded %d requests, want 1", len(stored))
	}
}

// ─── The admin queue ──────────────────────────────────────────────────────────

func TestDeletionQueue_IsOpsOnly(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	for _, role := range []string{"konsultan", "tour_leader"} {
		h.as(role).GET("/api/v1/admin/privacy/deletion-requests").
			expectCode(http.StatusForbidden)
	}
	for _, role := range []string{"admin", "super_admin"} {
		h.as(role).GET("/api/v1/admin/privacy/deletion-requests").
			expectCode(http.StatusOK)
	}
}

// Approving does the erasure §25.5 promises, and leaves standing what §25.4 says
// to keep: the participant row for statistics, and the invoices as financial
// records.
func TestDeletionRequest_ApprovalAnonymisesAndRemovesDocuments(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request",
		map[string]any{"reason": "sudah tidak dipakai"}).expectCode(http.StatusOK)
	stored, _ := h.Deletions.List(context.Background(), "")
	id := stored[0].ID

	h.as("admin").POST("/api/v1/admin/privacy/deletion-requests/"+id+"/process",
		map[string]any{"decision": "setujui", "notes": "tidak ada keberangkatan aktif"}).
		expectCode(http.StatusOK)

	p, err := h.Participants.GetByID(context.Background(), "participant-1")
	if err != nil {
		t.Fatalf("participant disappeared entirely — §25.4 anonymises rather than deletes: %v", err)
	}
	if p.Name == "Peserta Satu" || p.Email != "" {
		t.Errorf("identifying fields survived: %+v", p)
	}
	if p.IsActive {
		t.Error("an anonymised participant is still active")
	}

	docs, err := h.Documents.ListByParticipant(context.Background(), "participant-1")
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("%d documents survived the erasure — the scans are the sensitive part", len(docs))
	}

	// The invoice is a financial record and stays readable.
	if _, err := h.Invoices.GetByID(context.Background(), "invoice-1"); err != nil {
		t.Errorf("the invoice was removed with the personal data: %v", err)
	}

	after, _ := h.Deletions.List(context.Background(), "")
	if after[0].Status != privacy.StatusDone || after[0].ProcessedBy == nil {
		t.Errorf("request closed as %+v, want done with an actor", after[0])
	}
}

// A refusal is a decision that has to be answerable later, so it records who and
// why — and touches nothing.
func TestDeletionRequest_RejectionKeepsTheDataAndRecordsWhy(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request", nil).
		expectCode(http.StatusOK)
	stored, _ := h.Deletions.List(context.Background(), "")

	h.as("admin").POST("/api/v1/admin/privacy/deletion-requests/"+stored[0].ID+"/process",
		map[string]any{"decision": "tolak", "notes": "keberangkatan H-7, paspor masih dibutuhkan"}).
		expectCode(http.StatusOK)

	p, _ := h.Participants.GetByID(context.Background(), "participant-1")
	if p.Name != "Peserta Satu" {
		t.Error("a rejected request still anonymised the participant")
	}
	docs, _ := h.Documents.ListByParticipant(context.Background(), "participant-1")
	if len(docs) != 2 {
		t.Errorf("%d documents left after a rejection, want both", len(docs))
	}

	after, _ := h.Deletions.List(context.Background(), "")
	if after[0].Status != privacy.StatusRejected || after[0].Notes == "" {
		t.Errorf("rejection recorded as %+v, want a reason", after[0])
	}
}

// A refusal without a reason is not answerable, so it is refused.
func TestDeletionRequest_RejectionRequiresAReason(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request", nil).
		expectCode(http.StatusOK)
	stored, _ := h.Deletions.List(context.Background(), "")

	h.as("admin").POST("/api/v1/admin/privacy/deletion-requests/"+stored[0].ID+"/process",
		map[string]any{"decision": "tolak"}).
		expectCode(http.StatusBadRequest)
}

// Two reviewers working the same queue: the second is told the decision was
// already taken rather than being allowed to overturn or repeat it.
func TestDeletionRequest_CannotBeProcessedTwice(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request", nil).
		expectCode(http.StatusOK)
	stored, _ := h.Deletions.List(context.Background(), "")
	path := "/api/v1/admin/privacy/deletion-requests/" + stored[0].ID + "/process"

	h.as("admin").POST(path, map[string]any{"decision": "setujui"}).expectCode(http.StatusOK)
	h.as("super_admin").POST(path, map[string]any{"decision": "tolak", "notes": "berubah pikiran"}).
		expectCode(http.StatusConflict)
}

// The queue shows how long each request has been waiting, because the 14-day
// commitment is measured in exactly that.
func TestDeletionQueue_ReportsHowLongEachHasWaited(t *testing.T) {
	h := newHarness(t)
	seedErasureCandidate(h)

	h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request", nil).
		expectCode(http.StatusOK)

	res := h.as("admin").GET("/api/v1/admin/privacy/deletion-requests?status=menunggu")
	res.expectCode(http.StatusOK)

	var payload struct {
		Data []struct {
			DaysWaiting *int `json:"days_waiting"`
			IsOpen      bool `json:"is_open"`
		} `json:"data"`
		Meta map[string]int `json:"meta"`
	}
	res.decode(&payload)
	if len(payload.Data) != 1 {
		t.Fatalf("queue returned %d rows, want one pending request", len(payload.Data))
	}
	if payload.Data[0].DaysWaiting == nil {
		t.Error("queue entry does not say how long it has waited")
	}
	if !payload.Data[0].IsOpen {
		t.Error("a pending request is not reported as open")
	}
}
