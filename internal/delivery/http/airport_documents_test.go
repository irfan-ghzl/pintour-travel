package httpdelivery

// Airport report + document pagination (tiket 10).

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	domainDocument "github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

// ─── The read that used to write ──────────────────────────────────────────────

// The criterion this ticket names. The checklist dashboard polls this endpoint
// every ten seconds, and it used to initialise the whole batch's checklist rows
// on the way past — one open tab ran a write across every participant six times
// a minute, with the error thrown away.
func TestListChecklist_DoesNotWrite(t *testing.T) {
	h := newHarness(t)
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-1", BatchID: "batch-1", Name: "Peserta Satu", IsActive: true,
	})

	before := len(h.Airport.checklists)
	h.as("tour_leader").GET("/api/v1/admin/airport/checklist?batch_id=batch-1").
		expectCode(http.StatusOK)

	if after := len(h.Airport.checklists); after != before {
		t.Errorf("membuka daftar checklist menulis data: %d → %d baris", before, after)
	}
}

// Initialisation still happens, as its own action, and says so when it fails.
func TestInitChecklist_IsAnExplicitActionThatReportsFailure(t *testing.T) {
	h := newHarness(t)

	h.as("tour_leader").POST("/api/v1/admin/airport/checklist/init",
		map[string]string{"batch_id": "batch-1"}).expectCode(http.StatusOK)

	h.Airport.Fail(errWriteFailed)
	h.as("tour_leader").POST("/api/v1/admin/airport/checklist/init",
		map[string]string{"batch_id": "batch-1"}).
		expectCode(http.StatusInternalServerError)
}

// A summary that cannot be read is an error, not a silent row of zeros — a tour
// leader would hand that report over believing it.
func TestAirportReport_FailsLoudlyWhenTheSummaryIsUnavailable(t *testing.T) {
	h := newHarness(t)
	h.Airport.Fail(errWriteFailed)

	h.as("tour_leader").GET("/api/v1/admin/airport/report/pdf?batch_id=batch-1").
		expectCode(http.StatusInternalServerError)
}

// ─── Document pagination ──────────────────────────────────────────────────────

func seedManyDocuments(h *harness, n int) {
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-1", BatchID: "batch-1", Name: "Peserta Satu", IsActive: true,
	})
	for i := 0; i < n; i++ {
		status := "menunggu"
		if i%2 == 0 {
			status = "disetujui"
		}
		h.Documents.Seed(domainDocument.Document{
			ID:            fmt.Sprintf("document-%02d", i),
			ParticipantID: "participant-1", DocumentType: "passport",
			FilePath: "participant-1/passport.jpg", FileName: "passport.jpg",
			Status: status, UploadedAt: time.Now(),
		})
	}
}

// Every other list in the system is paginated; this one fetched every document
// ever uploaded in order to show twenty.
func TestDocuments_ListIsPaginated(t *testing.T) {
	h := newHarness(t)
	seedManyDocuments(h, 45)

	res := h.as("admin").GET("/api/v1/admin/documents?per_page=20&page=1")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"meta"`
	}
	res.decode(&envelope)

	if len(envelope.Data) != 20 {
		t.Errorf("baris di halaman = %d, ingin 20", len(envelope.Data))
	}
	if envelope.Meta.Total != 45 {
		t.Errorf("meta.total = %d, ingin 45", envelope.Meta.Total)
	}
	if envelope.Meta.TotalPages != 3 {
		t.Errorf("meta.total_pages = %d, ingin 3", envelope.Meta.TotalPages)
	}
}

// The "N of M approved" figure describes all of a participant's documents, not
// the filter in front of them. Counting the rows on screen made it agree with
// itself and nothing else: filtering to approved showed "2 of 2".
func TestDocuments_ReviewSummaryCountsEveryDocumentOfTheParticipant(t *testing.T) {
	h := newHarness(t)
	seedManyDocuments(h, 10) // 5 disetujui, 5 menunggu

	res := h.as("admin").
		GET("/api/v1/admin/documents?participant_id=participant-1&status=disetujui")
	res.expectCode(http.StatusOK)

	var envelope struct {
		Summary struct {
			Total     int `json:"total"`
			Disetujui int `json:"disetujui"`
		} `json:"summary"`
	}
	res.decode(&envelope)

	if envelope.Summary.Total != 10 {
		t.Errorf("summary.total = %d, ingin 10 (seluruh dokumen peserta, bukan hasil filter)",
			envelope.Summary.Total)
	}
	if envelope.Summary.Disetujui != 5 {
		t.Errorf("summary.disetujui = %d, ingin 5", envelope.Summary.Disetujui)
	}
}

// The queue's "N of M approved" badge describes the participant, so it has to be
// right on the default view too — where no participant filter is set and the
// status filter hides most of their documents. It used to be counted from the
// rows on screen: someone two of three documents through read "0 of 1".
func TestDocumentQueue_ProgressDescribesTheParticipantNotThePage(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	// Three documents, two approved. The default queue shows only the pending one.
	for i, status := range []string{"disetujui", "disetujui", "menunggu"} {
		h.Documents.Seed(domainDocument.Document{
			ID:            fmt.Sprintf("document-progress-%d", i),
			ParticipantID: "participant-1",
			DocumentType:  []string{"passport", "ktp", "rekening_koran"}[i],
			FilePath:      "participant-1/berkas.jpg",
			FileName:      "berkas.jpg",
			Status:        status,
		})
	}

	res := h.as("admin").GET("/api/v1/admin/documents?status=menunggu")
	res.expectCode(http.StatusOK)

	var payload struct {
		Data      []map[string]any `json:"data"`
		Summaries map[string]struct {
			Total     int `json:"total"`
			Disetujui int `json:"disetujui"`
		} `json:"summaries"`
	}
	res.decode(&payload)

	if len(payload.Data) != 1 {
		t.Fatalf("queue shows %d rows, want only the pending document", len(payload.Data))
	}
	got, ok := payload.Summaries["participant-1"]
	if !ok {
		t.Fatal("no summary for the participant on the page — the badge would count the rows instead")
	}
	if got.Total != 3 || got.Disetujui != 2 {
		t.Errorf("summary = %d of %d approved, want 2 of 3 — the figure describes the participant, not the page",
			got.Disetujui, got.Total)
	}
}
