package httpdelivery

// PRD §19.3 promises that every user input is validated with go-playground/validator
// at the handler layer. These cases hold that promise to the letter: each one sends a
// value the schema does not know through a real route and expects a 400 naming the
// offending field, then sends a legitimate value through the same route and expects it
// to go through untouched. The second half matters as much as the first — validation
// that refuses payloads which used to work is a regression, not a feature.

import (
	"net/http"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
)

// ─── Staff roles (§5.3) ───────────────────────────────────────────────────────

func TestStaffRoleValidatedAgainstOfficialRoles(t *testing.T) {
	newStaff := func(role string) map[string]any {
		return map[string]any{
			"name":     "Staf Baru",
			"email":    "staf.baru@pintour.test",
			"password": "rahasia123",
			"role":     role,
		}
	}

	t.Run("tolak peran di luar §5.3", func(t *testing.T) {
		h := newHarness(t)
		before := len(h.Users.users)
		res := h.as("super_admin").POST("/api/v1/admin/users", newStaff("manajer"))
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("role")
		if len(h.Users.users) != before {
			t.Error("akun dengan peran tidak sah tetap tersimpan")
		}
	})

	t.Run("terima keempat peran resmi", func(t *testing.T) {
		for _, role := range staffRoles {
			h := newHarness(t)
			h.as("super_admin").POST("/api/v1/admin/users", newStaff(role)).
				expectCode(http.StatusCreated)
		}
	})

	t.Run("tolak peran di luar §5.3 saat menyunting", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("super_admin").PUT("/api/v1/admin/users/user-admin",
			map[string]any{"role": "manajer"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("role")
	})

	t.Run("terima penyuntingan tanpa mengubah peran", func(t *testing.T) {
		h := newHarness(t)
		h.as("super_admin").PUT("/api/v1/admin/users/user-admin",
			map[string]any{"name": "Nama Baru"}).expectCode(http.StatusOK)
	})
}

// ─── Email ────────────────────────────────────────────────────────────────────

func TestEmailFormatValidated(t *testing.T) {
	t.Run("tolak email tanpa bentuk alamat", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("super_admin").POST("/api/v1/admin/users", map[string]any{
			"name": "Staf Baru", "email": "bukan-email", "password": "rahasia123", "role": "admin",
		})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("email")
	})

	t.Run("terima email yang sah", func(t *testing.T) {
		h := newHarness(t)
		h.as("super_admin").POST("/api/v1/admin/users", map[string]any{
			"name": "Staf Baru", "email": "staf.baru@pintour.test", "password": "rahasia123", "role": "admin",
		}).expectCode(http.StatusCreated)
	})

	t.Run("tolak email tidak sah pada lupa password", func(t *testing.T) {
		h := newHarness(t)
		res := h.anonymous().POST("/api/v1/auth/forgot-password", map[string]any{"email": "@"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("email")
	})
}

// ─── Nomor telepon ────────────────────────────────────────────────────────────

func TestPhoneFormatValidated(t *testing.T) {
	newLead := func(phone string) map[string]any {
		return map[string]any{"name": "Prospek Baru", "phone": phone, "package_id": "package-1"}
	}

	t.Run("tolak nomor yang bukan nomor Indonesia", func(t *testing.T) {
		for _, phone := range []string{"12345", "0812", "+1 415 555 0100", "delapan"} {
			h := newHarness(t)
			h.seedBaseline()
			res := h.anonymous().POST("/api/v1/leads", newLead(phone))
			res.expectCode(http.StatusBadRequest)
			res.expectMessageMentions("phone")
		}
	})

	t.Run("terima ketiga bentuk yang dipakai formulir", func(t *testing.T) {
		for _, phone := range []string{"081234567890", "6281234567890", "+6281234567890"} {
			h := newHarness(t)
			h.seedBaseline()
			h.anonymous().POST("/api/v1/leads", newLead(phone)).expectCode(http.StatusCreated)
		}
	})

	t.Run("terima nomor bertanda pemisah seperti sebelum validasi ada", func(t *testing.T) {
		// The old CreateLead check judged the number after normalizePhone, which
		// leaves separators in place — so these were accepted, and must stay so.
		for _, phone := range []string{"0812-3456-7890", "0812 3456 7890", "+62 812-3456-7890"} {
			h := newHarness(t)
			h.seedBaseline()
			h.anonymous().POST("/api/v1/leads", newLead(phone)).expectCode(http.StatusCreated)
		}
	})

	t.Run("tolak nomor tidak sah pada login portal", func(t *testing.T) {
		h := newHarness(t)
		res := h.anonymous().POST("/api/v1/portal/login",
			map[string]any{"phone": "12345", "password": "rahasia123"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("phone")
	})
}

// ─── Jumlah pax ───────────────────────────────────────────────────────────────

func TestLeadPaxValidated(t *testing.T) {
	newLead := func(pax any) map[string]any {
		body := map[string]any{
			"name": "Prospek Baru", "phone": "081234567890", "package_id": "package-1",
		}
		if pax != nil {
			body["pax"] = pax
		}
		return body
	}

	t.Run("tolak jumlah di luar rentang skema", func(t *testing.T) {
		for _, pax := range []any{-1, 51, 1000} {
			h := newHarness(t)
			h.seedBaseline()
			res := h.anonymous().POST("/api/v1/leads", newLead(pax))
			res.expectCode(http.StatusBadRequest)
			res.expectMessageMentions("pax")
		}
	})

	t.Run("pax yang tidak diisi menjadi satu, bukan nol", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		h.anonymous().POST("/api/v1/leads", newLead(nil)).expectCode(http.StatusCreated)
		// The column's CHECK is BETWEEN 1 AND 50 and the repository writes this
		// field verbatim, so 0 would be a 500 against a real database.
		for _, l := range h.Leads.leads {
			if l.Name == "Prospek Baru" && l.Pax == 0 {
				t.Error("lead tersimpan dengan pax = 0")
			}
		}
	})

	t.Run("terima jumlah dalam rentang", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		h.anonymous().POST("/api/v1/leads", newLead(4)).expectCode(http.StatusCreated)
	})
}

// ─── Tipe kamar ───────────────────────────────────────────────────────────────

func TestRoomTypeValidated(t *testing.T) {
	// seedConvertible puts a lead in the one status ConvertFromLead accepts, with a
	// batch to convert it into.
	seedConvertible := func(h *harness) {
		h.seedBaseline()
		h.Leads.Seed(domainLead.Lead{
			ID: "lead-deal", Name: "Prospek Deal", Phone: "628111111112",
			PackageID: "package-1", Status: "deal", CreatedAt: time.Now(),
		})
		h.Batches.Seed(domainPkg.PackageBatch{
			ID: "batch-1", PackageID: "package-1", Quota: 40, Status: "tersedia",
			DepartureDate: time.Now().Add(30 * 24 * time.Hour),
		})
	}

	t.Run("tolak tipe kamar di luar skema", func(t *testing.T) {
		h := newHarness(t)
		seedConvertible(h)
		res := h.as("admin").POST("/api/v1/admin/participants/convert", map[string]any{
			"lead_id": "lead-deal", "batch_id": "batch-1", "room_type": "presidential",
		})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("room_type")
		if len(h.Participants.participants) != 1 { // hanya peserta dari seedBaseline
			t.Error("peserta dibuat dengan tipe kamar tidak sah")
		}
	})

	t.Run("terima ketiga tipe kamar yang dikenal skema", func(t *testing.T) {
		for _, roomType := range []string{"single", "double", "triple"} {
			h := newHarness(t)
			seedConvertible(h)
			h.as("admin").POST("/api/v1/admin/participants/convert", map[string]any{
				"lead_id": "lead-deal", "batch_id": "batch-1", "room_type": roomType,
			}).expectCode(http.StatusCreated)
		}
	})
}

// ─── Status invoice ───────────────────────────────────────────────────────────

func TestInvoiceStatusValidated(t *testing.T) {
	newInvoice := func(status string) map[string]any {
		body := map[string]any{
			"participant_id": "participant-1",
			"batch_id":       "batch-1",
			"amount":         25000000,
			"due_date":       time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		}
		if status != "" {
			body["status"] = status
		}
		return body
	}

	t.Run("tolak status di luar kosakata skema", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		res := h.as("admin").POST("/api/v1/admin/invoices", newInvoice("belum_bayar"))
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("status")
	})

	t.Run("terima kelima status yang dikenal skema", func(t *testing.T) {
		statuses := []string{"diterbitkan", "menunggu_bayar", "dibayar", "lunas", "menunggu_konfirmasi_gateway"}
		for _, status := range statuses {
			h := newHarness(t)
			h.seedBaseline()
			h.as("admin").POST("/api/v1/admin/invoices", newInvoice(status)).
				expectCode(http.StatusCreated)
		}
	})

	t.Run("terima payload tanpa status", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		h.as("admin").POST("/api/v1/admin/invoices", newInvoice("")).expectCode(http.StatusCreated)
	})

	t.Run("tolak status review bukti bayar di luar dua nilai yang sah", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		res := h.as("admin").PATCH("/api/v1/admin/invoices/invoice-1/proofs/proof-1/review",
			map[string]any{"status": "mungkin"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("status")
	})
}

// ─── Status dokumen ───────────────────────────────────────────────────────────

func TestDocumentStatusValidated(t *testing.T) {
	seedDoc := func(h *harness) {
		h.seedBaseline()
		h.Documents.Seed(document.Document{
			ID: "document-1", ParticipantID: "participant-1",
			DocumentType: "passport", FilePath: "docs/passport.jpg",
			FileName: "passport.jpg", Status: "menunggu",
		})
	}

	t.Run("tolak status review di luar dua nilai yang sah", func(t *testing.T) {
		h := newHarness(t)
		seedDoc(h)
		res := h.as("admin").PATCH("/api/v1/admin/documents/document-1/review",
			map[string]any{"status": "mungkin"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("status")
	})

	t.Run("terima kedua status review yang sah", func(t *testing.T) {
		h := newHarness(t)
		seedDoc(h)
		h.as("admin").PATCH("/api/v1/admin/documents/document-1/review",
			map[string]any{"status": "disetujui"}).expectCode(http.StatusOK)

		h.as("admin").PATCH("/api/v1/admin/documents/document-1/review",
			map[string]any{"status": "ditolak", "rejection_reason": "buram"}).expectCode(http.StatusOK)
	})

	t.Run("tolak jenis dokumen di luar kosakata skema", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		res := h.as("admin").POST("/api/v1/admin/participants/participant-1/documents",
			map[string]any{"document_type": "sim", "file_path": "docs/sim.jpg", "file_name": "sim.jpg"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("document_type")
	})

	t.Run("terima jenis dokumen yang dikenal skema", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		h.as("admin").POST("/api/v1/admin/participants/participant-1/documents",
			map[string]any{"document_type": "passport", "file_path": "docs/p.jpg", "file_name": "p.jpg"}).
			expectCode(http.StatusCreated)
	})
}

// ─── Status lead ──────────────────────────────────────────────────────────────

func TestLeadStatusValidated(t *testing.T) {
	t.Run("tolak status di luar pipeline §12", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		res := h.as("admin").PATCH("/api/v1/admin/leads/lead-1/status",
			map[string]any{"status": "selesai"})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("status")
	})

	t.Run("terima setiap status pipeline", func(t *testing.T) {
		for _, status := range domainLead.Statuses {
			h := newHarness(t)
			h.seedBaseline()
			h.as("admin").PATCH("/api/v1/admin/leads/lead-1/status",
				map[string]any{"status": status}).expectCode(http.StatusOK)
		}
	})
}

// ─── Katalog: kategori paket, status batch, persyaratan negara ────────────────

func TestCatalogVocabulariesValidated(t *testing.T) {
	newPackage := func(category string) map[string]any {
		return map[string]any{
			"name": "Paket Baru", "destination": "Jepang", "category": category,
			"duration_days": 7, "base_price": 30000000,
		}
	}
	newBatch := func(status string) map[string]any {
		body := map[string]any{
			"departure_date": time.Now().Add(60 * 24 * time.Hour).Format(time.RFC3339),
			"return_date":    time.Now().Add(67 * 24 * time.Hour).Format(time.RFC3339),
			"quota":          40, "price_single": 35000000,
			"price_double": 30000000, "price_triple": 28000000,
		}
		if status != "" {
			body["status"] = status
		}
		return body
	}

	t.Run("tolak kategori paket di luar skema", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("admin").POST("/api/v1/admin/packages", newPackage("backpacker"))
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("category")
	})

	t.Run("terima keempat kategori yang dikenal skema", func(t *testing.T) {
		for _, category := range []string{"reguler", "umroh", "halal", "honeymoon"} {
			h := newHarness(t)
			h.as("admin").POST("/api/v1/admin/packages", newPackage(category)).
				expectCode(http.StatusCreated)
		}
	})

	t.Run("tolak paket tanpa harga dasar", func(t *testing.T) {
		h := newHarness(t)
		body := newPackage("umroh")
		delete(body, "base_price")
		res := h.as("admin").POST("/api/v1/admin/packages", body)
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("base_price")
	})

	t.Run("tolak status batch di luar skema", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		res := h.as("admin").POST("/api/v1/admin/packages/package-1/batches", newBatch("libur"))
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("status")
	})

	t.Run("terima batch yang lengkap", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		h.as("admin").POST("/api/v1/admin/packages/package-1/batches", newBatch("tersedia")).
			expectCode(http.StatusCreated)
	})

	t.Run("tolak persyaratan negara dengan jenis dokumen tak dikenal", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("admin").POST("/api/v1/admin/country-requirements", map[string]any{
			"country_code": "JP", "country_name": "Jepang", "document_type": "sim",
		})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("document_type")
	})

	t.Run("terima persyaratan negara yang sah", func(t *testing.T) {
		h := newHarness(t)
		h.as("admin").POST("/api/v1/admin/country-requirements", map[string]any{
			"country_code": "JP", "country_name": "Jepang", "document_type": "passport",
		}).expectCode(http.StatusCreated)
	})
}

// ─── Batas paginasi ───────────────────────────────────────────────────────────

func TestPageSizeCapped(t *testing.T) {
	// Every paginated list shares one ceiling, so one route proves the rule and the
	// table below proves it is actually shared.
	paths := []string{
		"/api/v1/admin/invoices",
		"/api/v1/admin/leads",
		"/api/v1/admin/participants",
		"/api/v1/admin/packages",
		"/api/v1/admin/chatbot-logs",
	}

	t.Run("permintaan tak wajar dipangkas ke batas atas", func(t *testing.T) {
		for _, path := range paths {
			h := newHarness(t)
			h.seedBaseline()
			res := h.as("admin").GET(path + "?per_page=100000&limit=100000")
			res.expectCode(http.StatusOK)
			if limit := res.metaLimit(); limit > maxPerPage {
				t.Errorf("%s: meta.limit = %d, melebihi batas %d", path, limit, maxPerPage)
			}
		}
	})

	t.Run("permintaan wajar dilewatkan apa adanya", func(t *testing.T) {
		h := newHarness(t)
		h.seedBaseline()
		res := h.as("admin").GET("/api/v1/admin/invoices?per_page=50")
		res.expectCode(http.StatusOK)
		if limit := res.metaLimit(); limit != 50 {
			t.Errorf("meta.limit = %d, want 50", limit)
		}
	})
}

// ─── Bentuk galat ─────────────────────────────────────────────────────────────

func TestValidationErrorsAreBadRequestsNamingTheField(t *testing.T) {
	t.Run("galat validasi menyebut field, bukan galat internal", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("super_admin").POST("/api/v1/admin/users", map[string]any{
			"name": "Staf Baru", "email": "staf@pintour.test", "password": "pendek", "role": "admin",
		})
		res.expectCode(http.StatusBadRequest)
		res.expectMessageMentions("password")

		var envelope struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		res.decode(&envelope)
		if envelope.Success {
			t.Error("envelope success = true pada galat validasi")
		}
		if envelope.Error != "BAD_REQUEST" {
			t.Errorf("kode galat = %q, want BAD_REQUEST", envelope.Error)
		}
	})

	t.Run("beberapa field bermasalah semuanya disebut", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("super_admin").POST("/api/v1/admin/users", map[string]any{"phone": "12345"})
		res.expectCode(http.StatusBadRequest)
		for _, field := range []string{"name", "email", "password", "role", "phone"} {
			res.expectMessageMentions(field)
		}
	})

	t.Run("body yang bukan JSON dijawab pesan cadangan handler", func(t *testing.T) {
		h := newHarness(t)
		res := h.as("super_admin").POST("/api/v1/admin/users", "{bukan json")
		res.expectCode(http.StatusBadRequest)
		// Nothing decoded, so there is no field to name — the handler's own
		// message stands, unchanged from before validation existed.
		const fallback = "nama, email, password, dan role harus diisi"
		if msg := res.message(); msg != fallback {
			t.Errorf("pesan = %q, want cadangan %q", msg, fallback)
		}
	})
}
