package httpdelivery

import (
	"net/http"
	"testing"
	"time"

	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
)

// Daftar leads sudah disaring per konsultan sejak awal, dan penyaringan itulah
// yang membuat pembatasannya tampak selesai. Detailnya tidak — konsultan yang
// bukan pemilik cukup menyalin satu ID untuk membaca nama, nomor WhatsApp, dan
// isi pesan calon pelanggan milik rekannya.
//
// Ditemukan saat UAT terhadap sistem yang berjalan, bukan dari membaca kode:
// daftarnya benar, sehingga tidak ada yang mencurigakan sampai endpoint
// detailnya dipanggil langsung dengan ID milik orang lain.

// seedLeadOwnedBy menyemai satu lead yang sudah ditugaskan ke seorang konsultan.
func seedLeadOwnedBy(h *harness, leadID, consultantID string) {
	h.Leads.Seed(domainLead.Lead{
		ID: leadID, Name: "Prospek Milik Orang Lain", Phone: "628770000009",
		PackageID: "package-1", Status: "konsultasi", AssignedTo: &consultantID,
		CreatedAt: time.Now(),
	})
}

func TestLeadDetail_KonsultanCannotOpenSomeoneElsesLead(t *testing.T) {
	h := newHarness(t)
	seedLeadOwnedBy(h, "lead-milik-a", "user-konsultan")

	// 404, bukan 403: membedakan "ada tapi bukan milikmu" dari "tidak ada"
	// sendirian sudah memberi tahu ID mana yang terpakai.
	h.asUser("konsultan-b", "konsultan").
		GET("/api/v1/admin/leads/lead-milik-a").
		expectCode(http.StatusNotFound)
}

func TestLeadDetail_KonsultanStillOpensTheirOwnLead(t *testing.T) {
	h := newHarness(t)
	seedLeadOwnedBy(h, "lead-milik-a", "user-konsultan")

	h.as("konsultan").
		GET("/api/v1/admin/leads/lead-milik-a").
		expectCode(http.StatusOK)
}

// Pembatasan hanya berlaku untuk konsultan. Admin memegang seluruh CRM, dan
// perbaikan kebocoran di atas tidak boleh ikut menutup pintunya.
func TestLeadDetail_AdminOpensAnyLead(t *testing.T) {
	for _, role := range []string{"super_admin", "admin"} {
		h := newHarness(t)
		seedLeadOwnedBy(h, "lead-milik-a", "user-konsultan")

		h.as(role).
			GET("/api/v1/admin/leads/lead-milik-a").
			expectCode(http.StatusOK)
	}
}
