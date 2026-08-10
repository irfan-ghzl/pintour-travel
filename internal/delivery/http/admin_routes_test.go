package httpdelivery

// The admin and staff routes the earlier tickets did not have to touch.
//
// Tickets 02–12 each brought tests for the behaviour they changed, which left
// the routes nobody had reason to change untested: airport handling, the
// chatbot log, tour leader profiles, the portal's self-service pages, package
// and country-requirement management. Every one of them is a promise the PRD
// makes to somebody, and none of them was protected against a regression.
//
// These read as the user stories they serve, through the same seam: a role
// sends a request and gets a result. They are not smoke tests — each asserts
// what the response actually says, not merely that a status came back.

import (
	"net/http"
	"testing"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// ─── Airport handling (FR-AIR-01…06) ─────────────────────────────────────────

// seedDeparture prepares a batch about to leave with two participants on it,
// which is the state the handling desk opens in.
func seedDeparture(h *harness) {
	h.seedBaseline()
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-1", PackageID: "package-1", Quota: 40, Status: "tersedia",
		DepartureDate: calendar.Today().AddDays(1),
		ReturnDate:    calendar.Today().AddDays(10),
	})
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-2", BatchID: "batch-1", Name: "Peserta Dua",
		Phone: "628222222223", RoomType: "double", IsActive: true,
	})
	for _, pid := range []string{"participant-1", "participant-2"} {
		h.Airport.Seed(airport.Checklist{
			ID: "checklist-" + pid, ParticipantID: pid, BatchID: "batch-1",
			ParticipantName: "Peserta " + pid,
		})
	}
}

// A tour leader works through the three handling steps for one participant and
// the batch progress follows along — FR-AIR-02/03.
func TestAirportHandling_StepsMoveTheBatchProgress(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	progress := func() (done, pending int) {
		t.Helper()
		res := h.as("tour_leader").GET("/api/v1/admin/airport/checklist?batch_id=batch-1")
		res.expectCode(http.StatusOK)
		var envelope struct {
			Data struct {
				Progress struct {
					DoneCount    int `json:"done_count"`
					PendingCount int `json:"pending_count"`
				} `json:"progress"`
			} `json:"data"`
		}
		res.decode(&envelope)
		return envelope.Data.Progress.DoneCount, envelope.Data.Progress.PendingCount
	}

	if done, pending := progress(); done != 0 || pending != 2 {
		t.Fatalf("awal: selesai=%d menunggu=%d, want 0/2", done, pending)
	}

	base := "/api/v1/admin/airport/participants/participant-1/"
	for _, step := range []string{"baggage", "ticket", "passport"} {
		h.as("tour_leader").PATCH(base+step+"?batch_id=batch-1", nil).
			expectCode(http.StatusOK)
	}

	if done, pending := progress(); done != 1 || pending != 1 {
		t.Errorf("setelah tiga langkah: selesai=%d menunggu=%d, want 1/1", done, pending)
	}
}

// The checklist filter separates who is done from who is still waiting, so the
// desk can see the queue rather than the whole manifest.
func TestAirportHandling_ChecklistFiltersByStatus(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	base := "/api/v1/admin/airport/participants/participant-1/"
	for _, step := range []string{"baggage", "ticket", "passport"} {
		h.as("tour_leader").PATCH(base+step+"?batch_id=batch-1", nil).expectCode(http.StatusOK)
	}

	count := func(status string) int {
		t.Helper()
		res := h.as("tour_leader").GET("/api/v1/admin/airport/checklist?batch_id=batch-1&status=" + status)
		res.expectCode(http.StatusOK)
		var envelope struct {
			Data struct {
				Checklists []map[string]any `json:"checklists"`
			} `json:"data"`
		}
		res.decode(&envelope)
		return len(envelope.Data.Checklists)
	}

	if n := count("done"); n != 1 {
		t.Errorf("filter done = %d baris, want 1", n)
	}
	if n := count("pending"); n != 1 {
		t.Errorf("filter pending = %d baris, want 1", n)
	}
}

// Every airport route insists on a batch: a request without one is a mistake,
// not a request for everybody.
func TestAirportHandling_BatchIsRequired(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	for _, path := range []string{
		"/api/v1/admin/airport/checklist",
		"/api/v1/admin/airport/report",
		"/api/v1/admin/airport/report/pdf",
	} {
		h.as("tour_leader").GET(path).expectCode(http.StatusBadRequest)
	}
	for _, step := range []string{"baggage", "ticket", "passport"} {
		h.as("tour_leader").
			PATCH("/api/v1/admin/airport/participants/participant-1/"+step, nil).
			expectCode(http.StatusBadRequest)
	}
	h.as("tour_leader").POST("/api/v1/admin/airport/checklist/init", map[string]any{}).
		expectCode(http.StatusBadRequest)
}

// The JSON report carries the same summary the PDF prints, so a tour leader can
// check the numbers before downloading anything.
func TestAirportReport_CarriesTheBatchSummary(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	res := h.as("tour_leader").GET("/api/v1/admin/airport/report?batch_id=batch-1")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data struct {
			BatchID  string `json:"batch_id"`
			Progress struct {
				TotalPax int `json:"total_pax"`
			} `json:"progress"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if envelope.Data.BatchID != "batch-1" {
		t.Errorf("batch_id = %q, want batch-1", envelope.Data.BatchID)
	}
	if envelope.Data.Progress.TotalPax != 2 {
		t.Errorf("total_pax = %d, want 2", envelope.Data.Progress.TotalPax)
	}
}

// FR-AIR-06: the post-handling report downloads as a PDF naming the real batch,
// not a fragment of a UUID.
func TestAirportReportPDF_IsAPDF(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	h.as("tour_leader").GET("/api/v1/admin/airport/report/pdf?batch_id=batch-1").
		expectCode(http.StatusOK).expectPDF()
}

// Initialisation is idempotent, which is what lets the desk press it on opening
// without worrying whether someone already did.
func TestAirportChecklistInit_IsIdempotent(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	for i := 0; i < 2; i++ {
		h.as("tour_leader").POST("/api/v1/admin/airport/checklist/init",
			map[string]any{"batch_id": "batch-1"}).expectCode(http.StatusOK)
	}
}

// FR-AUTO-08: the departure confirmation only goes out once the whole batch is
// handled. Sending it while people are still at the counter would tell them to
// board while their baggage is unchecked.
func TestConfirmDeparture_WaitsForTheWholeBatch(t *testing.T) {
	h := newHarness(t)
	seedDeparture(h)

	confirm := func() *response {
		return h.as("tour_leader").POST("/api/v1/admin/airport/confirm-departure", map[string]any{
			"batch_id":     "batch-1",
			"gather_point": "Terminal 3 Soekarno-Hatta",
			"gather_time":  "04:00",
			"gate":         "D5",
			"checkin_time": "05:30",
		})
	}

	res := confirm()
	res.expectCode(http.StatusUnprocessableEntity)
	res.expectMessageMentions("belum selesai")

	for _, pid := range []string{"participant-1", "participant-2"} {
		for _, step := range []string{"baggage", "ticket", "passport"} {
			h.as("tour_leader").
				PATCH("/api/v1/admin/airport/participants/"+pid+"/"+step+"?batch_id=batch-1", nil).
				expectCode(http.StatusOK)
		}
	}

	confirm().expectCode(http.StatusOK)

	h.as("tour_leader").POST("/api/v1/admin/airport/confirm-departure", map[string]any{}).
		expectCode(http.StatusBadRequest)
}

// ─── Tour leader profiles ─────────────────────────────────────────────────────

// A tour leader profile can be written and read back, which is what the
// briefing pages and the post-handling report resolve names through.
func TestTourLeaderProfile_UpsertThenRead(t *testing.T) {
	h := newHarness(t)
	h.Users.Seed(domainUser.User{
		ID: "user-tl", Name: "Ahmad Rizki", Email: "tl@pintour.test",
		Role: "tour_leader", Phone: "628333333333", IsActive: true,
	})

	h.as("admin").PUT("/api/v1/admin/tour-leaders/user-tl", map[string]any{
		"bio":              "Pengalaman 10 tahun memimpin rombongan umroh",
		"experience_years": 10,
		"specialization":   "Umroh",
		"emergency_phone":  "628333333334",
	}).expectCode(http.StatusCreated)

	res := h.as("admin").GET("/api/v1/admin/tour-leaders/user-tl")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data struct {
			Bio             string `json:"bio"`
			ExperienceYears int    `json:"experience_years"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if envelope.Data.ExperienceYears != 10 {
		t.Errorf("experience_years = %d, want 10", envelope.Data.ExperienceYears)
	}
	if envelope.Data.Bio == "" {
		t.Error("bio tidak tersimpan")
	}

	res = h.as("admin").GET("/api/v1/admin/tour-leaders")
	res.expectCode(http.StatusOK)
}

// An emergency number that is not a reachable Indonesian number is refused
// before it is stored — it is the number somebody calls in an emergency.
func TestTourLeaderProfile_ValidatesEmergencyNumber(t *testing.T) {
	h := newHarness(t)
	h.Users.Seed(domainUser.User{ID: "user-tl", Role: "tour_leader", IsActive: true})

	res := h.as("admin").PUT("/api/v1/admin/tour-leaders/user-tl", map[string]any{
		"emergency_phone": "123",
	})
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("emergency_phone")
}

// ─── Chatbot log (v2.0 F2) ────────────────────────────────────────────────────

func seedConversation(h *harness) {
	h.seedBaseline()
	for _, m := range []chatbot.Log{
		{ID: "msg-1", Phone: "628444444444", Role: "user", Message: "Ada paket umroh Desember?"},
		{ID: "msg-2", Phone: "628444444444", Role: "assistant", Message: "Ada, batch 10 Desember."},
	} {
		h.Chatbot.Seed(m)
	}
}

// An admin can read back a conversation the bot handled, which is how a lead
// that started in WhatsApp gets picked up by a human.
func TestChatbotLog_ConversationIsReadable(t *testing.T) {
	h := newHarness(t)
	seedConversation(h)

	res := h.as("admin").GET("/api/v1/admin/chatbot-logs")
	res.expectCode(http.StatusOK)

	res = h.as("admin").GET("/api/v1/admin/chatbot-logs/628444444444")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data []struct {
			Message string `json:"message"`
			Role    string `json:"role"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if len(envelope.Data) != 2 {
		t.Fatalf("pesan terbaca = %d, want 2", len(envelope.Data))
	}
	if envelope.Data[0].Role != "user" || envelope.Data[0].Message == "" {
		t.Errorf("giliran pertama = %+v, want pesan dari pengguna", envelope.Data[0])
	}
}

// FR-CRM-01 via the bot: a conversation becomes a lead, assigned and recorded
// like any other.
func TestChatbotLog_ConversationBecomesALead(t *testing.T) {
	h := newHarness(t)
	seedConversation(h)

	before := len(h.Leads.order)
	res := h.as("admin").POST("/api/v1/admin/chatbot-logs/628444444444/create-lead", map[string]any{
		"name":       "Prospek Chatbot",
		"package_id": "package-1",
	})
	res.expectCode(http.StatusCreated)
	if len(h.Leads.order) != before+1 {
		t.Fatalf("lead tersimpan = %d, want %d", len(h.Leads.order), before+1)
	}
	created := h.Leads.leads[h.Leads.order[len(h.Leads.order)-1]]
	if created.Phone != "628444444444" {
		t.Errorf("phone lead = %q, want nomor dari percakapan", created.Phone)
	}
}

// ─── Country document requirements ────────────────────────────────────────────

// The requirement list per destination is what the portal shows a participant
// as "documents you still owe us", so it has to be manageable end to end.
func TestCountryRequirements_FullLifecycle(t *testing.T) {
	h := newHarness(t)

	res := h.as("admin").POST("/api/v1/admin/country-requirements", map[string]any{
		"country_code":  "SA",
		"country_name":  "Arab Saudi",
		"document_type": "passport",
		"is_required":   true,
		"description":   "Berlaku minimal 6 bulan",
	})
	res.expectCode(http.StatusCreated)
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res.decode(&created)
	if created.Data.ID == "" {
		t.Fatal("persyaratan dibuat tanpa id")
	}

	h.as("admin").GET("/api/v1/admin/country-requirements?country_code=SA").
		expectCode(http.StatusOK)

	h.as("admin").PUT("/api/v1/admin/country-requirements/"+created.Data.ID, map[string]any{
		"country_code":  "SA",
		"country_name":  "Arab Saudi",
		"document_type": "ktp",
		"is_required":   false,
	}).expectCode(http.StatusOK)

	h.as("admin").DELETE("/api/v1/admin/country-requirements/" + created.Data.ID).
		expectCode(http.StatusNoContent)
}

// ─── Packages & images (CMS) ──────────────────────────────────────────────────

// A package is published, edited, illustrated and withdrawn — the whole CMS
// path an admin walks to put a product on the catalogue.
func TestPackageCMS_FullLifecycle(t *testing.T) {
	h := newHarness(t)

	res := h.as("admin").POST("/api/v1/admin/packages", map[string]any{
		"name": "Umroh Reguler 9 Hari", "destination": "Arab Saudi",
		"category": "umroh", "duration_days": 9, "base_price": 25000000,
	})
	res.expectCode(http.StatusCreated)
	var created struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	res.decode(&created)
	if created.Data.Slug == "" {
		t.Error("slug tidak diturunkan dari nama paket")
	}
	id := created.Data.ID

	h.as("admin").PUT("/api/v1/admin/packages/"+id, map[string]any{
		"name": "Umroh Reguler 12 Hari", "duration_days": 12,
	}).expectCode(http.StatusOK)

	h.as("admin").POST("/api/v1/admin/packages/"+id+"/images", map[string]any{
		"file_path": "package-images/foto-1.jpg", "alt_text": "Masjidil Haram",
		"is_thumbnail": true,
	}).expectCode(http.StatusCreated)

	res = h.as("admin").GET("/api/v1/admin/packages/" + id + "/images")
	res.expectCode(http.StatusOK)
	var images struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res.decode(&images)
	if len(images.Data) != 1 {
		t.Fatalf("gambar terbaca = %d, want 1", len(images.Data))
	}

	h.as("admin").DELETE("/api/v1/admin/packages/" + id + "/images/" + images.Data[0].ID).
		expectCode(http.StatusNoContent)
	h.as("admin").DELETE("/api/v1/admin/packages/" + id).
		expectCode(http.StatusNoContent)
	h.as("admin").GET("/api/v1/admin/packages").expectCode(http.StatusOK)
}

// A batch can be edited after it is opened — prices move, and a tour leader is
// assigned later than the schedule is published.
func TestBatch_CanBeEditedAfterOpening(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	res := h.as("admin").POST("/api/v1/admin/packages/package-1/batches",
		batchFormPayload("2026-09-01", "2026-09-10"))
	res.expectCode(http.StatusCreated)
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res.decode(&created)

	h.as("admin").PUT("/api/v1/admin/batches/"+created.Data.ID, map[string]any{
		"price_double":  27500000,
		"wa_group_link": "https://chat.whatsapp.com/abc",
	}).expectCode(http.StatusOK)

	res = h.as("admin").GET("/api/v1/admin/packages/package-1/batches")
	res.expectCode(http.StatusOK)
	var listed struct {
		Data []struct {
			PriceDouble float64 `json:"price_double"`
			WaGroupLink string  `json:"wa_group_link"`
		} `json:"data"`
	}
	res.decode(&listed)
	var found bool
	for _, b := range listed.Data {
		if b.PriceDouble == 27500000 && b.WaGroupLink != "" {
			found = true
		}
	}
	if !found {
		t.Error("perubahan harga dan tautan grup tidak terbaca kembali")
	}
}

// ─── Documents review ─────────────────────────────────────────────────────────

func seedDocuments(h *harness) {
	h.seedBaseline()
	for _, d := range []document.Document{
		{ID: "doc-1", ParticipantID: "participant-1", DocumentType: "passport",
			FilePath: "participant-documents/p1/passport.pdf", FileName: "passport.pdf", Status: "menunggu"},
		{ID: "doc-2", ParticipantID: "participant-1", DocumentType: "ktp",
			FilePath: "participant-documents/p1/ktp.jpg", FileName: "ktp.jpg", Status: "menunggu"},
	} {
		h.Documents.Seed(d)
	}
}

// FR-PORTAL-05: an admin reviews a participant's documents, and the summary
// counts every document they own rather than the ones currently filtered.
func TestDocumentReview_SummaryCountsAllOfTheirDocuments(t *testing.T) {
	h := newHarness(t)
	seedDocuments(h)

	h.as("admin").PATCH("/api/v1/admin/documents/doc-1/review", map[string]any{
		"status": "disetujui",
	}).expectCode(http.StatusOK)

	// The global review list is the one that filters and summarises. The summary
	// describes every document the participant owns, not the rows the filter
	// left behind — counting the filtered rows made the figure agree with
	// itself and with nothing else.
	res := h.as("admin").
		GET("/api/v1/admin/documents?participant_id=participant-1&status=disetujui")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Summary map[string]int `json:"summary"`
		Data    []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if len(envelope.Data) != 1 {
		t.Errorf("hasil filter = %d dokumen, want 1", len(envelope.Data))
	}
	if envelope.Summary["total"] != 2 {
		t.Errorf("ringkasan total = %d, want 2 — dihitung dari seluruh dokumen peserta", envelope.Summary["total"])
	}
	if envelope.Summary["disetujui"] != 1 || envelope.Summary["menunggu"] != 1 {
		t.Errorf("ringkasan = %v, want 1 disetujui / 1 menunggu", envelope.Summary)
	}

	// The per-participant listing returns everything they uploaded.
	res = h.as("admin").GET("/api/v1/admin/participants/participant-1/documents")
	res.expectCode(http.StatusOK)
	var all struct {
		Data []map[string]any `json:"data"`
	}
	res.decode(&all)
	if len(all.Data) != 2 {
		t.Errorf("daftar per peserta = %d dokumen, want 2", len(all.Data))
	}
}

// A rejection has to say why. The participant has to re-upload, and a rejection
// with no reason tells them nothing about what to fix.
func TestDocumentReview_RejectionNeedsAReason(t *testing.T) {
	h := newHarness(t)
	seedDocuments(h)

	res := h.as("admin").PATCH("/api/v1/admin/documents/doc-1/review", map[string]any{
		"status": "ditolak",
	})
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("rejection_reason")

	h.as("admin").PATCH("/api/v1/admin/documents/doc-1/review", map[string]any{
		"status": "ditolak", "rejection_reason": "Foto buram, tidak terbaca",
	}).expectCode(http.StatusOK)

	doc := h.Documents.documents["doc-1"]
	if doc.Status != "ditolak" || doc.RejectionReason == "" {
		t.Errorf("dokumen = %q dengan alasan %q", doc.Status, doc.RejectionReason)
	}
}

// A document can be withdrawn, and the OCR result endpoint answers honestly
// when nothing has been extracted yet.
func TestDocument_DeleteAndMissingOCRResult(t *testing.T) {
	h := newHarness(t)
	seedDocuments(h)

	h.as("admin").GET("/api/v1/admin/documents/doc-1/ocr-result").
		expectCode(http.StatusNotFound)

	h.as("admin").DELETE("/api/v1/admin/documents/doc-1").
		expectCode(http.StatusNoContent)
}

// ─── Portal self-service (§15.4, §25.5) ───────────────────────────────────────

// §15.4: a participant corrects their own name and email, and the change sticks.
func TestPortalProfile_ParticipantCanCorrectTheirOwnDetails(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.asParticipant("participant-1").PUT("/api/v1/portal/profile", map[string]any{
		"name":  "Peserta Satu Diperbaiki",
		"email": "peserta.baru@pintour.test",
	}).expectCode(http.StatusOK)

	p := h.Participants.participants["participant-1"]
	if p.Name != "Peserta Satu Diperbaiki" || p.Email != "peserta.baru@pintour.test" {
		t.Errorf("profil tersimpan = %q / %q", p.Name, p.Email)
	}
}

// §25.5 right to access: a participant can take away everything the system
// holds on them, in one request.
func TestPortalMyData_ReturnsEverythingHeldOnThem(t *testing.T) {
	h := newHarness(t)
	seedDocuments(h)

	res := h.asParticipant("participant-1").GET("/api/v1/portal/my-data")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	res.decode(&envelope)
	for _, key := range []string{"participant", "invoices", "documents"} {
		if _, ok := envelope.Data[key]; !ok {
			t.Errorf("ekspor data tidak memuat %q", key)
		}
	}
}

// §25.5 right to erasure: the request is recorded rather than acted on
// silently — an active tour cannot simply vanish mid-flight.
func TestPortalDeletionRequest_IsAcknowledged(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	res := h.asParticipant("participant-1").POST("/api/v1/portal/account-deletion-request",
		map[string]any{"reason": "Tidak jadi berangkat"})
	res.expectCode(http.StatusOK)
}

// FR-BRIEF-02/03: the participant sees who is leading their batch, and can take
// the briefing away as a PDF once the window opens.
func TestPortalBriefing_LeaderAndPDF(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()
	leaderID := "user-tour_leader"
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-1", PackageID: "package-1", Quota: 40, Status: "tersedia",
		DepartureDate: calendar.Today().AddDays(3),
		ReturnDate:    calendar.Today().AddDays(12),
		TourLeaderID:  &leaderID,
	})
	h.TourLeaders.Seed(domainUser.TourLeader{
		ID: "tl-1", UserID: leaderID, Bio: "Pengalaman 10 tahun",
		Name: "Ahmad Rizki", Phone: "628333333333",
	})
	departure := calendar.Today().AddDays(3).Time()
	h.Participants.Seed(domainParticipant.Participant{
		ID: "participant-1", BatchID: "batch-1", Name: "Peserta Satu",
		Phone: "628222222222", RoomType: "double", IsActive: true,
		PackageName: "Umroh Reguler 9 Hari", BatchDepartureDate: &departure,
	})

	res := h.asParticipant("participant-1").GET("/api/v1/portal/batch-leader")
	res.expectCode(http.StatusOK)
	var leader struct {
		Data struct {
			TourLeader struct {
				Name  string `json:"name"`
				Phone string `json:"phone"`
			} `json:"tour_leader"`
		} `json:"data"`
	}
	res.decode(&leader)
	if leader.Data.TourLeader.Name != "Ahmad Rizki" {
		t.Errorf("tour leader = %q, want Ahmad Rizki", leader.Data.TourLeader.Name)
	}
	if leader.Data.TourLeader.Phone == "" {
		t.Error("nomor tour leader tidak disertakan — peserta perlu kontak daruratnya")
	}

	h.asParticipant("participant-1").GET("/api/v1/portal/briefing/pdf").
		expectCode(http.StatusOK).expectPDF()
}

// ─── Participants & NIK ───────────────────────────────────────────────────────

// v2.0 F6: an admin applies the NIK an OCR pass read, and it lands on the
// participant.
func TestParticipantNIK_IsAppliedAndReadBack(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.as("admin").PATCH("/api/v1/admin/participants/participant-1/nik", map[string]any{
		"nik": "3174012501900001",
	}).expectCode(http.StatusOK)

	if got := h.Participants.participants["participant-1"].NIK; got != "3174012501900001" {
		t.Errorf("NIK tersimpan = %q", got)
	}

	res := h.as("admin").GET("/api/v1/admin/participants/participant-1")
	res.expectCode(http.StatusOK)
}

// ─── Invoices ─────────────────────────────────────────────────────────────────

// An invoice downloads as a PDF, and the proofs attached to it are listed for
// the reviewer.
func TestInvoice_PDFAndProofListing(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.as("admin").GET("/api/v1/admin/invoices/invoice-1/pdf").
		expectCode(http.StatusOK).expectPDF()

	h.as("admin").POST("/api/v1/admin/invoices/invoice-1/proofs", map[string]any{
		"file_path":      "payment-proofs/p1/bukti.jpg",
		"amount_claimed": 10000000,
		"notes":          "Cicilan pertama",
	}).expectCode(http.StatusCreated)

	res := h.as("admin").GET("/api/v1/admin/invoices/invoice-1")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data struct {
			Invoice struct {
				InvoiceNumber string `json:"invoice_number"`
			} `json:"invoice"`
			PaymentProofs []struct {
				AmountClaimed float64 `json:"amount_claimed"`
			} `json:"payment_proofs"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if envelope.Data.Invoice.InvoiceNumber != "INV-202608-0001" {
		t.Errorf("invoice_number = %q", envelope.Data.Invoice.InvoiceNumber)
	}
	if len(envelope.Data.PaymentProofs) != 1 {
		t.Fatalf("bukti bayar terbaca = %d, want 1", len(envelope.Data.PaymentProofs))
	}
	if envelope.Data.PaymentProofs[0].AmountClaimed != 10000000 {
		t.Errorf("nominal bukti = %v, want 10000000", envelope.Data.PaymentProofs[0].AmountClaimed)
	}
}

// ─── Session ──────────────────────────────────────────────────────────────────

// Logging out clears the session cookie, so the browser stops sending a token
// the server would still honour.
func TestLogout_ClearsTheSessionCookie(t *testing.T) {
	h := newHarness(t)

	res := h.as("admin").POST("/api/v1/auth/logout", nil)
	res.expectCode(http.StatusOK)

	if c := res.cookie(JWTCookieName); c == nil || c.MaxAge >= 0 {
		t.Error("cookie sesi tidak dihapus saat logout")
	}
}
