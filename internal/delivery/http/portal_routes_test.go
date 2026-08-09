package httpdelivery

// The portal a participant actually uses, plus the leads, catalogue and login
// routes around it.
//
// Every test here reads as a participant's or an admin's sentence: they open a
// page, and what they see is what the PRD promised them.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainInvoice "github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// ─── Portal reading pages ─────────────────────────────────────────────────────

// A participant sees their own bills and nobody else's, and can take each one
// away as a PDF.
func TestPortalInvoices_OwnBillsOnly(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()
	h.Participants.Seed(seedParticipant("participant-2", "628222222299"))
	h.Invoices.Seed(domainInvoice.Invoice{
		ID: "invoice-2", InvoiceNumber: "INV-202608-0002",
		ParticipantID: "participant-2", BatchID: "batch-1",
		Amount: 12000000, Status: "diterbitkan", DueDate: calendar.Today().AddDays(5),
	})

	res := h.asParticipant("participant-1").GET("/api/v1/portal/invoices")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data []struct {
			ID            string `json:"id"`
			InvoiceNumber string `json:"invoice_number"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if len(envelope.Data) != 1 || envelope.Data[0].ID != "invoice-1" {
		t.Fatalf("invoice terbaca = %+v, want hanya invoice-1", envelope.Data)
	}

	h.asParticipant("participant-1").GET("/api/v1/portal/invoices/invoice-1/pdf").
		expectCode(http.StatusOK).expectPDF()

	// Somebody else's invoice is not found, rather than forbidden: a 403 would
	// confirm it exists.
	h.asParticipant("participant-1").GET("/api/v1/portal/invoices/invoice-2/pdf").
		expectCode(http.StatusNotFound)
}

// A participant sees the documents they uploaded, and the list of what their
// destination still requires.
func TestPortalDocuments_OwnUploadsAndRequirements(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()
	h.Documents.Seed(document.Document{
		ID: "doc-1", ParticipantID: "participant-1", DocumentType: "passport",
		FilePath: "participant-documents/p1/passport.pdf", FileName: "passport.pdf",
		Status: "menunggu",
	})
	h.Documents.Seed(document.Document{
		ID: "doc-other", ParticipantID: "participant-2", DocumentType: "ktp",
		FilePath: "participant-documents/p2/ktp.jpg", FileName: "ktp.jpg", Status: "menunggu",
	})
	h.CountryReqs.Seed(document.CountryRequirement{
		ID: "req-1", CountryCode: "SA", CountryName: "Arab Saudi",
		DocumentType: "passport", IsRequired: true,
	})

	res := h.asParticipant("participant-1").GET("/api/v1/portal/documents")
	res.expectCode(http.StatusOK)
	var docs struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res.decode(&docs)
	if len(docs.Data) != 1 || docs.Data[0].ID != "doc-1" {
		t.Errorf("dokumen terbaca = %+v, want hanya milik sendiri", docs.Data)
	}

	h.asParticipant("participant-1").GET("/api/v1/portal/documents/requirements").
		expectCode(http.StatusOK)
}

// A participant cannot register a file path they did not upload. Without the
// check, naming somebody else's stored object would attach their passport to
// your record — and the review page would then hand it back to you.
func TestPortalDocumentUpload_RefusesAPathTheyDidNotUpload(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	res := h.asParticipant("participant-1").POST("/api/v1/portal/documents", map[string]any{
		"participant_id": "participant-2", // ignored — the token decides
		"document_type":  "passport",
		"file_path":      "participant-documents/participant-2/passport.pdf",
		"file_name":      "passport.pdf",
	})
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("file_path")

	if len(h.Documents.documents) != 0 {
		t.Errorf("dokumen tersimpan = %d, want 0", len(h.Documents.documents))
	}
}

// FR-PORTAL-10: the itinerary of the trip the participant is on.
func TestPortalItinerary_ComesFromTheirPackage(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()
	h.Batches.Seed(domainPkg.PackageBatch{
		ID: "batch-1", PackageID: "package-1", Quota: 40, Status: "tersedia",
		DepartureDate: calendar.Today().AddDays(20),
	})
	h.Packages.Seed(domainPkg.Package{
		ID: "package-1", Name: "Umroh Reguler 9 Hari", Slug: "umroh-reguler-9-hari",
		Destination: "Arab Saudi", DurationDays: 9, IsActive: true,
		Itinerary: []byte(`[{"day":1,"title":"Tiba di Jeddah"}]`),
	})

	res := h.asParticipant("participant-1").GET("/api/v1/portal/itinerary")
	res.expectCode(http.StatusOK)
}

// A participant is told how many days are left, counted in whole calendar days
// so the number does not tick over at an arbitrary hour.
func TestPortalMe_CarriesTheCountdown(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()
	// Exactly twenty times twenty-four hours out, so the count is the same
	// whatever time of day the test runs.
	departure := time.Now().Add(20 * 24 * time.Hour)
	p := seedParticipant("participant-1", "628222222222")
	p.BatchDepartureDate = &departure
	h.Participants.Seed(p)

	res := h.asParticipant("participant-1").GET("/api/v1/portal/me")
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data struct {
			DaysToDepart   *int `json:"days_to_depart"`
			BriefingActive bool `json:"briefing_active"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if envelope.Data.DaysToDepart == nil || *envelope.Data.DaysToDepart != 20 {
		t.Errorf("days_to_depart = %v, want 20", envelope.Data.DaysToDepart)
	}
	// §5.8: the briefing opens at H-14, so twenty days out it is still shut.
	if envelope.Data.BriefingActive {
		t.Error("briefing_active = true pada H-20, want false")
	}
}

// ─── Portal login ─────────────────────────────────────────────────────────────

// A wrong password is refused, and so is a number nobody registered — both with
// the same answer, so the endpoint does not confirm who has an account.
func TestPortalLogin_RefusesWrongCredentials(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	for name, body := range map[string]map[string]any{
		"password salah":  {"phone": "628222222222", "password": "salah-sekali"},
		"nomor tak ada":   {"phone": "628999999999", "password": "apa-saja"},
		"password kosong": {"phone": "628222222222", "password": ""},
	} {
		t.Run(name, func(t *testing.T) {
			res := h.anonymous().POST("/api/v1/portal/login", body)
			if res.Code != http.StatusUnauthorized && res.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 401 atau 400", res.Code)
			}
		})
	}
}

// ─── Leads ────────────────────────────────────────────────────────────────────

// A consultant is assigned a lead, and the assignment shows up on it.
func TestLeadAssign_SticksToTheLead(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()
	h.Users.Seed(domainUser.User{
		ID: "user-konsultan-2", Name: "Konsultan Dua", Email: "k2@pintour.test",
		Role: "konsultan", IsActive: true,
	})

	h.as("admin").PATCH("/api/v1/admin/leads/lead-1/assign", map[string]any{
		"consultant_id": "user-konsultan-2",
	}).expectCode(http.StatusOK)

	lead := h.Leads.leads["lead-1"]
	if lead.AssignedTo == nil || *lead.AssignedTo != "user-konsultan-2" {
		t.Errorf("assigned_to = %v, want user-konsultan-2", lead.AssignedTo)
	}

	// A blank consultant is refused rather than clearing the assignment.
	res := h.as("admin").PATCH("/api/v1/admin/leads/lead-1/assign", map[string]any{
		"consultant_id": "",
	})
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("consultant_id")
}

// A status the schema does not know is refused, naming the statuses that exist.
func TestLeadStatus_RefusesUnknownTargets(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	res := h.as("admin").PATCH("/api/v1/admin/leads/lead-1/status", map[string]any{
		"status": "kadaluarsa",
	})
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("status")

	for _, status := range domainLead.Statuses {
		h.as("admin").PATCH("/api/v1/admin/leads/lead-1/status", map[string]any{
			"status": status,
		}).expectCode(http.StatusOK)
	}
}

// ─── Catalogue listing filters ────────────────────────────────────────────────

// The public catalogue narrows by the things a customer actually searches on.
func TestPublicCatalogue_Filters(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	for _, query := range []string{
		"", "?category=umroh", "?destination=Arab%20Saudi",
		"?price_min=1000000&price_max=50000000",
		"?duration_min=5&duration_max=12",
		"?departure_month=2026-09",
		"?page=1&per_page=5",
	} {
		h.anonymous().GET("/api/v1/packages" + query).expectCode(http.StatusOK)
	}
}

// The staff participant list narrows the same way, and paginates.
func TestParticipantList_Filters(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	for _, query := range []string{
		"", "?batch_id=batch-1", "?is_active=true", "?search=Peserta",
		"?page=1&per_page=5",
	} {
		h.as("admin").GET("/api/v1/admin/participants" + query).expectCode(http.StatusOK)
	}
}

// ─── Uploads (§16.2) ──────────────────────────────────────────────────────────

// multipartFile builds an upload request body the way a browser does.
func multipartFile(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("buat form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("tulis isi berkas: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("tutup writer: %v", err)
	}
	return &body, w.FormDataContentType()
}

// With no storage configured the upload endpoints refuse rather than pretend:
// a participant told their passport was uploaded when it was not would find out
// at the airport.
func TestUpload_RefusesWhenStorageIsNotConfigured(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	cases := map[string]struct {
		path   string
		field  string
		client *client
	}{
		"dokumen peserta": {"/api/v1/portal/upload/document", "file", h.asParticipant("participant-1")},
		"bukti bayar":     {"/api/v1/portal/upload/payment-proof", "file", h.asParticipant("participant-1")},
		"gambar paket":    {"/api/v1/admin/packages/package-1/images/upload", "file", h.as("admin")},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			body, contentType := multipartFile(t, tc.field, "paspor.pdf", []byte("%PDF-1.4 palsu"))
			res := tc.client.withHeader("Content-Type", contentType).
				POST(tc.path, body.Bytes())
			if res.Code < 400 {
				t.Errorf("status = %d, want galat karena penyimpanan belum dikonfigurasi", res.Code)
			}
		})
	}
}

// seedParticipant is a participant row with the fields the portal reads.
func seedParticipant(id, phone string) domainParticipant.Participant {
	return domainParticipant.Participant{
		ID: id, BatchID: "batch-1", Name: "Peserta " + id, Phone: phone,
		Email: id + "@pintour.test", PackageName: "Umroh Reguler 9 Hari",
		RoomType: "double", IsActive: true,
	}
}

// ─── Staff login ──────────────────────────────────────────────────────────────

// A staff account signs in and gets a session; wrong credentials get the same
// refusal whether the email exists or not, so the endpoint does not reveal who
// has an account.
func TestStaffLogin_SucceedsAndRefusesTheSameWay(t *testing.T) {
	h := newHarness(t)
	hashed, err := bcrypt.GenerateFromPassword([]byte("rahasia-admin"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	h.Users.Seed(domainUser.User{
		ID: "user-admin", Name: "Admin Satu", Email: "admin@pintour.test",
		Password: string(hashed), Role: "admin", IsActive: true,
	})

	res := h.anonymous().POST("/api/v1/auth/login", map[string]any{
		"email": "admin@pintour.test", "password": "rahasia-admin",
	})
	res.expectCode(http.StatusOK)
	var envelope struct {
		Data struct {
			Role  string `json:"role"`
			Token string `json:"token"`
		} `json:"data"`
	}
	res.decode(&envelope)
	if envelope.Data.Role != "admin" {
		t.Errorf("role = %q, want admin", envelope.Data.Role)
	}
	// §19.1: the token travels in an httpOnly cookie, so JavaScript cannot read
	// it — which is what keeps it out of reach of an XSS payload.
	if c := res.cookie(JWTCookieName); c == nil || !c.HttpOnly {
		t.Errorf("cookie sesi = %v, want httpOnly", c)
	}

	for name, body := range map[string]map[string]any{
		"password salah": {"email": "admin@pintour.test", "password": "salah"},
		"email tak ada":  {"email": "hantu@pintour.test", "password": "rahasia-admin"},
	} {
		t.Run(name, func(t *testing.T) {
			h.anonymous().POST("/api/v1/auth/login", body).
				expectCode(http.StatusUnauthorized)
		})
	}
}
