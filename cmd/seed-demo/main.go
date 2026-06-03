// Package main — Seed dummy data lengkap untuk semua 14 tabel.
// Idempotent: aman dijalankan berkali-kali (pakai ON CONFLICT).
//
// Usage:
//
//	go run ./cmd/seed-demo           # insert/update dummy data
//	go run ./cmd/seed-demo -fresh    # WIPE + insert (DELETE seluruh data dulu)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()
	fresh := flag.Bool("fresh", false, "Hapus semua data sebelum insert (DANGEROUS)")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://pintour:pintour_pass@localhost:5432/pintour_db?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	must(err, "open db")
	defer db.Close()
	must(db.Ping(), "ping db")

	ctx := context.Background()

	if *fresh {
		fmt.Println("🗑️  Wiping all data (fresh mode)...")
		wipeAll(ctx, db)
	}

	pass := bcryptHash("admin123")
	passPeserta := bcryptHash("peserta123")

	// 1. USERS
	fmt.Println("→ Seeding users...")
	superAdminID := upsertUser(ctx, db, "admin@pintour.com", "Admin Pintour", pass, "super_admin", "628100000001")
	adminID := upsertUser(ctx, db, "operator@pintour.com", "Sri Operator", pass, "admin", "628100000002")
	konsultanID1 := upsertUser(ctx, db, "rina@pintour.com", "Rina Sales", pass, "konsultan", "628100000003")
	konsultanID2 := upsertUser(ctx, db, "budi@pintour.com", "Budi Sales", pass, "konsultan", "628100000004")
	tlID1 := upsertUser(ctx, db, "ahmad@pintour.com", "Ahmad Tour Leader", pass, "tour_leader", "628100000005")
	tlID2 := upsertUser(ctx, db, "siti@pintour.com", "Siti Tour Leader", pass, "tour_leader", "628100000006")
	_ = adminID

	// 2. TOUR_LEADERS profile
	fmt.Println("→ Seeding tour_leaders profile...")
	upsertTourLeader(ctx, db, tlID1, "Tour leader berpengalaman 10 tahun spesialis Eropa & Asia",
		"https://i.pravatar.cc/200?img=12", 10, "Eropa, Jepang, Korea", "628100000005")
	upsertTourLeader(ctx, db, tlID2, "Tour leader umroh & wisata halal Timur Tengah, fluent Arab",
		"https://i.pravatar.cc/200?img=23", 8, "Umroh, Arab Saudi, Turki, Dubai", "628100000006")

	// 3. COUNTRY_DOCUMENT_REQUIREMENTS
	fmt.Println("→ Seeding country_document_requirements...")
	insertCountryReqs(ctx, db)

	// 4. PACKAGES
	fmt.Println("→ Seeding packages...")
	pkgUmroh := insertPackage(ctx, db, packageSeed{
		Name: "Umroh Plus Istanbul 12 Hari", Slug: "umroh-plus-istanbul-12-hari",
		Destination: "Arab Saudi & Turki", Category: "umroh", Duration: 12,
		Description: "Paket umroh lengkap dengan tambahan wisata Istanbul. Termasuk hotel bintang 5 dekat Masjidil Haram, tiket pesawat PP, manasik, mutawif berpengalaman.",
		BasePrice:   29500000,
		Thumbnail:   "https://images.unsplash.com/photo-1565552645632-d725f8bfc19a?w=800",
		Facilities: []string{
			"Hotel Bintang 5 dekat Haramain",
			"Tiket Pesawat PP (Garuda Indonesia)",
			"Visa Umroh & Tourist Turki",
			"Manasik Umroh 2x sebelum keberangkatan",
			"Mutawif berpengalaman fluent Bahasa Indonesia",
			"Makan 3x sehari (menu Indonesia)",
			"Transportasi AC selama di Saudi & Turki",
			"Air zam-zam 5L",
		},
		Requirements: []string{
			"Paspor minimal masa berlaku 7 bulan",
			"Pas foto 4x6 background putih (5 lembar)",
			"Surat keterangan sehat",
			"Vaksin meningitis (sertifikat ICV)",
			"Buku nikah (untuk pasangan suami-istri)",
		},
		VisaInfo:        "Visa umroh diurus tim Pintour (sudah termasuk paket). Visa transit Turki single-entry diurus terpisah, dokumen disiapkan tim.",
		TermsConditions: "DP minimum Rp 5 juta saat booking. Pelunasan paling lambat H-30 keberangkatan. Pembatalan H-14 dikenakan biaya 50%.",
		CreatedBy:       superAdminID,
	})
	pkgJepang := insertPackage(ctx, db, packageSeed{
		Name: "Sakura Tour Jepang 8 Hari", Slug: "sakura-tour-jepang-8-hari",
		Destination: "Jepang", Category: "reguler", Duration: 8,
		Description: "Nikmati keindahan bunga sakura di Tokyo, Kyoto, dan Osaka. Naik Shinkansen, kunjungan Gunung Fuji, dan free time di Shibuya.",
		BasePrice:   24800000,
		Thumbnail:   "https://images.unsplash.com/photo-1522383225653-ed111181a951?w=800",
		Facilities: []string{
			"Hotel Bintang 4 (Tokyo, Kyoto, Osaka)",
			"Tiket Pesawat PP (ANA)",
			"Tiket Shinkansen Kyoto-Tokyo",
			"Local guide bilingual",
			"Sarapan + 4x makan siang",
			"Tiket masuk semua obyek wisata",
		},
		Requirements: []string{
			"Paspor minimal masa berlaku 6 bulan",
			"Rekening koran 3 bulan terakhir min. Rp 30 juta",
			"Slip gaji 3 bulan",
			"Surat sponsor (jika bukan karyawan)",
		},
		VisaInfo:        "Visa Jepang Single Entry. Dokumen diurus tim Pintour, processing 5-7 hari kerja. Biaya visa Rp 500.000 (sudah termasuk).",
		TermsConditions: "Booking ditutup H-30. Wajib ikut briefing sebelum keberangkatan.",
		CreatedBy:       superAdminID,
	})
	pkgTurki := insertPackage(ctx, db, packageSeed{
		Name: "Turki Wisata Halal 10 Hari", Slug: "turki-wisata-halal-10-hari",
		Destination: "Turki", Category: "halal", Duration: 10,
		Description: "Eksplorasi Turki: Istanbul, Cappadocia (hot air balloon!), Pamukkale, Bursa. Semua restoran halal, hotel muslim-friendly, ada waktu sholat berjamaah.",
		BasePrice:   21500000,
		Thumbnail:   "https://images.unsplash.com/photo-1524231757912-21f4fe3a7200?w=800",
		Facilities: []string{
			"Hotel Bintang 4 muslim-friendly",
			"Hot Air Balloon Cappadocia",
			"Tiket pesawat PP (Turkish Airlines)",
			"Visa elektronik",
			"Makan 3x sehari (semua halal)",
			"Tour leader fluent Bahasa Indonesia + Turki",
		},
		Requirements: []string{
			"Paspor masa berlaku min. 6 bulan",
			"Foto digital untuk e-visa",
		},
		VisaInfo:        "E-Visa Turki processing 5 hari. Sudah termasuk paket.",
		TermsConditions: "Kuota terbatas 25 pax per batch. DP 30% saat booking.",
		CreatedBy:       superAdminID,
	})
	pkgKorea := insertPackage(ctx, db, packageSeed{
		Name: "Korea Selatan Honeymoon 7 Hari", Slug: "korea-honeymoon-7-hari",
		Destination: "Korea Selatan", Category: "honeymoon", Duration: 7,
		Description: "Paket romantis untuk pasangan: Seoul, Nami Island, Petite France, Everland. Hotel double bed dengan view, dinner romantis di N Seoul Tower.",
		BasePrice:   18900000,
		Thumbnail:   "https://images.unsplash.com/photo-1538485399081-7c8978e8255b?w=800",
		Facilities: []string{
			"Hotel Bintang 4 (kamar honeymoon)",
			"Tiket pesawat PP",
			"Romantic dinner di N Seoul Tower",
			"Tiket Everland + Lotte World",
			"K-drama location tour",
			"Hanbok experience 1 jam",
		},
		Requirements: []string{
			"Paspor masa berlaku min. 6 bulan",
			"Surat nikah (untuk paket honeymoon)",
			"Rekening koran 3 bulan",
		},
		VisaInfo:        "Visa Korea single entry, processing 7-10 hari. Diurus tim Pintour.",
		TermsConditions: "Khusus pasangan suami-istri yang sudah menikah.",
		CreatedBy:       superAdminID,
	})
	pkgDubai := insertPackage(ctx, db, packageSeed{
		Name: "Dubai City Tour 5 Hari", Slug: "dubai-city-tour-5-hari",
		Destination: "UEA", Category: "reguler", Duration: 5,
		Description: "Pengalaman mewah di Dubai: Burj Khalifa, Desert Safari, Dubai Mall, Palm Jumeirah, Marina. Cocok untuk first-timer.",
		BasePrice:   15500000, Thumbnail: "https://images.unsplash.com/photo-1512453979798-5ea266f8880c?w=800",
		Facilities: []string{
			"Hotel Bintang 4 di Downtown Dubai",
			"Tiket Burj Khalifa lantai 124",
			"Desert Safari + BBQ dinner",
			"Tiket pesawat PP (Emirates)",
			"Visa UAE 30 hari",
		},
		Requirements: []string{
			"Paspor masa berlaku min. 6 bulan",
			"Foto digital untuk visa",
		},
		VisaInfo:        "Visa UAE diurus tim Pintour, processing 3-5 hari.",
		TermsConditions: "Paket termasuk visa & asuransi.",
		CreatedBy:       superAdminID,
	})

	// 5. PACKAGE_IMAGES
	fmt.Println("→ Seeding package_images...")
	insertImages(ctx, db, pkgUmroh, []string{
		"https://images.unsplash.com/photo-1565552645632-d725f8bfc19a?w=800",
		"https://images.unsplash.com/photo-1564507592333-c60657eea523?w=800",
		"https://images.unsplash.com/photo-1524231757912-21f4fe3a7200?w=800",
	})
	insertImages(ctx, db, pkgJepang, []string{
		"https://images.unsplash.com/photo-1522383225653-ed111181a951?w=800",
		"https://images.unsplash.com/photo-1545569341-9eb8b30979d9?w=800",
		"https://images.unsplash.com/photo-1493780474015-ba834fd0ce2f?w=800",
	})
	insertImages(ctx, db, pkgTurki, []string{
		"https://images.unsplash.com/photo-1524231757912-21f4fe3a7200?w=800",
		"https://images.unsplash.com/photo-1604941487773-7f1e6dfff7d1?w=800",
	})
	insertImages(ctx, db, pkgKorea, []string{
		"https://images.unsplash.com/photo-1538485399081-7c8978e8255b?w=800",
		"https://images.unsplash.com/photo-1517154421773-0529f29ea451?w=800",
	})
	insertImages(ctx, db, pkgDubai, []string{
		"https://images.unsplash.com/photo-1512453979798-5ea266f8880c?w=800",
	})

	// 6. PACKAGE_BATCHES
	fmt.Println("→ Seeding package_batches...")
	batchUmroh := insertBatch(ctx, db, pkgUmroh, daysFromNow(45), daysFromNow(57),
		25, 35000000, 29500000, 27500000, tlID2,
		"https://chat.whatsapp.com/abcd1234umroh")
	batchUmroh2 := insertBatch(ctx, db, pkgUmroh, daysFromNow(90), daysFromNow(102),
		30, 35000000, 29500000, 27500000, tlID2, "")
	batchJepang := insertBatch(ctx, db, pkgJepang, daysFromNow(60), daysFromNow(67),
		20, 28000000, 24800000, 23000000, tlID1,
		"https://chat.whatsapp.com/efgh5678jepang")
	batchTurki := insertBatch(ctx, db, pkgTurki, daysFromNow(75), daysFromNow(84),
		25, 25000000, 21500000, 20000000, tlID2, "")
	batchKorea := insertBatch(ctx, db, pkgKorea, daysFromNow(30), daysFromNow(36),
		15, 22000000, 18900000, 17500000, tlID1, "")
	batchDubai := insertBatch(ctx, db, pkgDubai, daysFromNow(20), daysFromNow(24),
		18, 18000000, 15500000, 14500000, tlID1, "")
	_ = batchUmroh2
	_ = batchDubai

	// 7. LEADS (various statuses)
	fmt.Println("→ Seeding leads...")
	insertLead(ctx, db, "Indra Permana", "628111000001", "indra@example.com",
		pkgUmroh, &batchUmroh, 2, "Mau tanya jadwal umroh bulan Juli", "meta_ads", "baru", nil)
	insertLead(ctx, db, "Dewi Lestari", "628111000002", "dewi@example.com",
		pkgJepang, &batchJepang, 1, "Tertarik paket Jepang, mau solo trip", "organic", "dihubungi", &konsultanID1)
	insertLead(ctx, db, "Ahmad Hidayat", "628111000003", "", pkgTurki, &batchTurki, 4,
		"Family trip, ada anak 8 tahun", "referral", "konsultasi", &konsultanID2)
	leadDeal := insertLead(ctx, db, "Sari Wulandari", "628111000004", "sari@example.com",
		pkgKorea, &batchKorea, 2, "Honeymoon, butuh hotel romantis", "meta_ads", "deal", &konsultanID1)
	insertLead(ctx, db, "Bayu Prasetyo", "628111000005", "", pkgDubai, &batchDubai, 3,
		"Family 3 orang, mau lihat Burj Khalifa", "walk_in", "tidak_deal", &konsultanID2)
	leadPeserta := insertLead(ctx, db, "Rini Anggraini", "628111000006", "rini@example.com",
		pkgUmroh, &batchUmroh, 2, "Berdua dengan ibu", "organic", "peserta", &konsultanID1)
	insertLead(ctx, db, "Joko Susanto", "628111000007", "joko@example.com",
		pkgJepang, &batchJepang, 1, "Mau lihat sakura", "meta_ads", "baru", nil)

	// 8. LEAD_NOTES
	fmt.Println("→ Seeding lead_notes...")
	insertLeadNote(ctx, db, leadDeal, konsultanID1, "Sudah call jam 10:00, peserta tertarik. Akan transfer DP minggu depan.")
	insertLeadNote(ctx, db, leadDeal, konsultanID1, "[SISTEM] Status diubah menjadi 'konsultasi'")
	insertLeadNote(ctx, db, leadDeal, konsultanID1, "[SISTEM] Status diubah menjadi 'deal'")
	insertLeadNote(ctx, db, leadDeal, konsultanID1, "Konfirmasi via email, dokumen visa sudah dikirim.")

	// 9. PARTICIPANTS (peserta aktif)
	fmt.Println("→ Seeding participants...")
	pPeserta1 := insertParticipant(ctx, db, &leadPeserta, batchUmroh, "Rini Anggraini", "628111000006",
		"rini@example.com", "double", passPeserta, true, false)
	pPeserta2 := insertParticipant(ctx, db, nil, batchKorea, "Mira Santoso", "628111000010",
		"mira@example.com", "double", passPeserta, true, true) // briefing viewed
	pPeserta3 := insertParticipant(ctx, db, nil, batchDubai, "Hendro Wijaya", "628111000011",
		"hendro@example.com", "single", passPeserta, true, false)
	pPeserta4 := insertParticipant(ctx, db, nil, batchJepang, "Putri Maharani", "628111000012",
		"putri@example.com", "triple", passPeserta, false, false) // belum aktif (belum bayar)

	// 10. INVOICES
	fmt.Println("→ Seeding invoices...")
	invLunas := insertInvoice(ctx, db, pPeserta1, batchUmroh, 29500000,
		daysFromNow(7), "lunas", superAdminID, &superAdminID, true)
	invDibayar := insertInvoice(ctx, db, pPeserta2, batchKorea, 18900000,
		daysFromNow(5), "dibayar", superAdminID, nil, false)
	invMenunggu := insertInvoice(ctx, db, pPeserta3, batchDubai, 15500000,
		daysFromNow(3), "menunggu_bayar", adminID, nil, false)
	invDiterbitkan := insertInvoice(ctx, db, pPeserta4, batchJepang, 24800000,
		daysFromNow(14), "diterbitkan", adminID, nil, false)
	_ = invDiterbitkan

	// 11. PAYMENT_PROOFS
	fmt.Println("→ Seeding payment_proofs...")
	insertPaymentProof(ctx, db, invLunas, 29500000, "Transfer BCA atas nama Rini Anggraini",
		"disetujui", &superAdminID, "Verified")
	insertPaymentProof(ctx, db, invDibayar, 18900000, "Transfer Mandiri",
		"menunggu", nil, "")
	insertPaymentProof(ctx, db, invMenunggu, 15000000, "Cicilan pertama (sisanya menyusul)",
		"menunggu", nil, "")

	// 12. DOCUMENTS
	fmt.Println("→ Seeding documents...")
	insertDocument(ctx, db, pPeserta1, "passport", "Paspor_Rini.pdf",
		"disetujui", "", &superAdminID)
	insertDocument(ctx, db, pPeserta1, "ktp", "KTP_Rini.jpg",
		"disetujui", "", &superAdminID)
	insertDocument(ctx, db, pPeserta1, "rekening_koran", "Rekening_Rini_3bulan.pdf",
		"menunggu", "", nil)
	insertDocument(ctx, db, pPeserta2, "passport", "Paspor_Mira.pdf",
		"disetujui", "", &superAdminID)
	insertDocument(ctx, db, pPeserta2, "ktp", "KTP_Mira.jpg",
		"ditolak", "Foto buram, harap upload ulang dengan resolusi lebih baik", &superAdminID)
	insertDocument(ctx, db, pPeserta3, "passport", "Paspor_Hendro.pdf",
		"menunggu", "", nil)

	// 13. AIRPORT_CHECKLISTS
	fmt.Println("→ Seeding airport_checklists...")
	insertAirportChecklist(ctx, db, pPeserta1, batchUmroh, false, false, false, &tlID2)
	insertAirportChecklist(ctx, db, pPeserta2, batchKorea, true, true, false, &tlID1)

	// 14. WA_NOTIFICATIONS (log untuk leads & invoice)
	fmt.Println("→ Seeding wa_notifications...")
	insertWANotif(ctx, db, "628111000004", "Sari Wulandari", "LEADS_WELCOME",
		"Halo Sari! Terima kasih sudah tertarik dengan paket Korea Honeymoon...", "lead", leadDeal, "sent")
	insertWANotif(ctx, db, "628100000003", "Rina Sales", "LEADS_NOTIF_SALES",
		"Leads baru: Sari Wulandari (628111000004), paket Korea Honeymoon", "lead", leadDeal, "sent")
	insertWANotif(ctx, db, "628111000004", "Sari Wulandari", "LEADS_WELCOME",
		"[konfirmasi] Status diubah ke 'konsultasi'", "lead", leadDeal, "sent")
	insertWANotif(ctx, db, "628111000006", "Rini Anggraini", "INVOICE_SENT",
		"Invoice Umroh Plus Istanbul sudah diterbitkan. Total Rp 29.500.000", "invoice", invLunas, "sent")
	insertWANotif(ctx, db, "628111000010", "Mira Santoso", "DOC_REQUEST",
		"Pembayaran terkonfirmasi! Silakan upload dokumen di portal", "participant", pPeserta2, "sent")
	insertWANotif(ctx, db, "628111000010", "Mira Santoso", "DOC_REJECTED",
		"Dokumen KTP Anda ditolak: foto buram", "document", pPeserta2, "sent")

	fmt.Println("\n✅ Seed selesai!")
	printSummary(ctx, db)
	fmt.Println("\n─────────────────────────────────────")
	fmt.Println("  🔑 Login Admin / Konsultan / TL")
	fmt.Println("  Email   : admin@pintour.com")
	fmt.Println("  Password: admin123")
	fmt.Println()
	fmt.Println("  🔑 Login Portal Peserta")
	fmt.Println("  Nomor WA: 628111000006 (Rini Anggraini)")
	fmt.Println("  Password: peserta123")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("  Frontend: http://localhost:3000 (atau :80 docker)")
	fmt.Println("  API     : http://localhost:8080/api/v1")
	fmt.Println("  Swagger : http://localhost:8080/swagger/index.html")
}

// ──── helpers ────────────────────────────────────────────────────────────────

func wipeAll(ctx context.Context, db *sql.DB) {
	// Order matters (FK constraint)
	tables := []string{
		"wa_notifications", "airport_checklists", "documents", "payment_proofs",
		"invoices", "lead_notes", "participants", "leads",
		"package_images", "package_batches", "packages",
		"country_document_requirements", "tour_leaders",
	}
	for _, t := range tables {
		_, err := db.ExecContext(ctx, "DELETE FROM "+t)
		if err != nil {
			log.Printf("wipe %s: %v", t, err)
		}
	}
	// Don't wipe users (keep admin), but reset to known state
	_, _ = db.ExecContext(ctx, `DELETE FROM users WHERE email IN
		('operator@pintour.com','rina@pintour.com','budi@pintour.com','ahmad@pintour.com','siti@pintour.com')`)
}

func upsertUser(ctx context.Context, db *sql.DB, email, name, hashedPass, role, phone string) string {
	var id string
	err := db.QueryRowContext(ctx, `
		INSERT INTO users (id, name, email, password, role, phone, is_active, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, true, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET
			name=$1, password=$3, role=$4, phone=$5, is_active=true, updated_at=NOW()
		RETURNING id`, name, email, hashedPass, role, phone).Scan(&id)
	must(err, "upsert user "+email)
	return id
}

func upsertTourLeader(ctx context.Context, db *sql.DB, userID, bio, photo string, years int, spec, emerg string) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO tour_leaders (id, user_id, bio, photo_path, experience_years, specialization, emergency_phone, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id) DO UPDATE SET bio=$2, photo_path=$3, experience_years=$4, specialization=$5, emergency_phone=$6`,
		userID, bio, photo, years, spec, emerg)
	must(err, "upsert tour_leader")
}

func insertCountryReqs(ctx context.Context, db *sql.DB) {
	reqs := []struct {
		code, name, docType, desc string
		required                  bool
	}{
		// Arab Saudi (umroh)
		{"SA", "Arab Saudi", "passport", "Paspor minimal masa berlaku 7 bulan", true},
		{"SA", "Arab Saudi", "ktp", "KTP", true},
		{"SA", "Arab Saudi", "visa_support", "Sertifikat vaksin meningitis (ICV)", true},
		{"SA", "Arab Saudi", "lainnya", "Buku nikah (untuk pasangan suami-istri)", false},
		// Jepang
		{"JP", "Jepang", "passport", "Paspor minimal masa berlaku 6 bulan", true},
		{"JP", "Jepang", "ktp", "KTP", true},
		{"JP", "Jepang", "rekening_koran", "Rekening koran 3 bulan terakhir min. Rp 30 juta", true},
		{"JP", "Jepang", "visa_support", "Slip gaji 3 bulan", true},
		{"JP", "Jepang", "lainnya", "Surat sponsor (jika bukan karyawan tetap)", false},
		// Turki
		{"TR", "Turki", "passport", "Paspor masa berlaku min. 6 bulan", true},
		{"TR", "Turki", "ktp", "KTP", true},
		// Korea
		{"KR", "Korea Selatan", "passport", "Paspor masa berlaku min. 6 bulan", true},
		{"KR", "Korea Selatan", "ktp", "KTP", true},
		{"KR", "Korea Selatan", "rekening_koran", "Rekening koran 3 bulan terakhir", true},
		{"KR", "Korea Selatan", "visa_support", "Surat nikah (untuk paket honeymoon)", false},
		// UAE
		{"AE", "Uni Emirat Arab", "passport", "Paspor masa berlaku min. 6 bulan", true},
		{"AE", "Uni Emirat Arab", "ktp", "KTP", true},
	}
	for _, r := range reqs {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO country_document_requirements (id, country_code, country_name, document_type, is_required, description, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW())`,
			r.code, r.name, r.docType, r.required, r.desc)
	}
}

type packageSeed struct {
	Name, Slug, Destination, Category, Description, Thumbnail string
	Duration                                                  int
	BasePrice                                                 float64
	Facilities, Requirements                                  []string
	VisaInfo, TermsConditions                                 string
	CreatedBy                                                 string
}

func insertPackage(ctx context.Context, db *sql.DB, p packageSeed) string {
	facJSON, _ := json.Marshal(p.Facilities)
	reqJSON, _ := json.Marshal(p.Requirements)
	itinJSON, _ := json.Marshal(generateItinerary(p.Duration, p.Destination))

	var id string
	err := db.QueryRowContext(ctx, `
		INSERT INTO packages (id, name, slug, destination, category, duration_days,
			description, itinerary, requirements, facilities, terms_conditions, visa_info,
			base_price, is_active, thumbnail, created_by, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9::jsonb, $10, $11, $12, true, $13, $14, NOW(), NOW())
		ON CONFLICT (slug) DO UPDATE SET
			name=$1, destination=$3, category=$4, duration_days=$5,
			description=$6, itinerary=$7::jsonb, requirements=$8::jsonb, facilities=$9::jsonb,
			terms_conditions=$10, visa_info=$11, base_price=$12, thumbnail=$13, updated_at=NOW()
		RETURNING id`,
		p.Name, p.Slug, p.Destination, p.Category, p.Duration,
		p.Description, string(itinJSON), string(reqJSON), string(facJSON), p.TermsConditions, p.VisaInfo,
		p.BasePrice, p.Thumbnail, p.CreatedBy).Scan(&id)
	must(err, "insert package "+p.Slug)
	return id
}

func generateItinerary(days int, destination string) []map[string]interface{} {
	out := []map[string]interface{}{}
	for i := 1; i <= days; i++ {
		title := fmt.Sprintf("Hari %d", i)
		desc := fmt.Sprintf("Aktivitas hari ke-%d di %s", i, destination)
		acts := []string{"Sarapan", "City tour", "Makan siang", "Free time"}
		switch i {
		case 1:
			title = "Hari 1: Keberangkatan & Tiba"
			desc = "Penjemputan di bandara, makan malam, check-in hotel"
			acts = []string{"Airport pickup", "Welcome dinner", "Hotel check-in", "Briefing tour"}
		case days:
			title = fmt.Sprintf("Hari %d: Kepulangan", i)
			desc = "Sarapan, check-out, transfer ke bandara, terbang pulang"
			acts = []string{"Breakfast", "Hotel check-out", "Last shopping", "Airport drop-off"}
		}
		out = append(out, map[string]interface{}{
			"day": i, "title": title, "description": desc, "activities": acts,
		})
	}
	return out
}

func insertImages(ctx context.Context, db *sql.DB, packageID string, urls []string) {
	for i, url := range urls {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO package_images (id, package_id, file_path, alt_text, sort_order, is_thumbnail, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW())`,
			packageID, url, fmt.Sprintf("Gambar %d", i+1), i, i == 0)
	}
}

func insertBatch(ctx context.Context, db *sql.DB, packageID string, departure, returnDate time.Time,
	quota int, single, double, triple float64, tourLeaderID, waGroupLink string) string {
	var id string
	var tlID interface{}
	if tourLeaderID != "" {
		tlID = tourLeaderID
	}
	err := db.QueryRowContext(ctx, `
		INSERT INTO package_batches (id, package_id, departure_date, return_date, quota,
			price_single, price_double, price_triple, status, tour_leader_id, wa_group_link, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, 'tersedia', $8, $9, NOW(), NOW())
		RETURNING id`,
		packageID, departure, returnDate, quota, single, double, triple, tlID, waGroupLink).Scan(&id)
	must(err, "insert batch")
	return id
}

func insertLead(ctx context.Context, db *sql.DB, name, phone, email, packageID string,
	batchID *string, pax int, message, source, status string, assignedTo *string) string {
	var id string
	convertedAt := interface{}(nil)
	if status == "peserta" {
		convertedAt = time.Now().Add(-7 * 24 * time.Hour)
	}
	err := db.QueryRowContext(ctx, `
		INSERT INTO leads (id, name, phone, email, package_id, batch_id, pax, message,
			source, status, assigned_to, converted_at, consent_given, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, NULLIF($3,''), $4, $5, $6, $7, $8, $9, $10, $11, true, NOW(), NOW())
		RETURNING id`,
		name, phone, email, packageID, batchID, pax, message, source, status, assignedTo, convertedAt).Scan(&id)
	must(err, "insert lead "+name)
	return id
}

func insertLeadNote(ctx context.Context, db *sql.DB, leadID, userID, note string) {
	_, _ = db.ExecContext(ctx, `
		INSERT INTO lead_notes (id, lead_id, user_id, note, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())`,
		leadID, userID, note)
}

func insertParticipant(ctx context.Context, db *sql.DB, leadID *string, batchID, name, phone, email,
	roomType, hashedPass string, isActive, briefingViewed bool) string {
	var id string
	err := db.QueryRowContext(ctx, `
		INSERT INTO participants (id, lead_id, batch_id, name, phone, email, room_type,
			portal_password, is_active, briefing_viewed, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NULLIF($5,''), $6, $7, $8, $9, NOW(), NOW())
		RETURNING id`,
		leadID, batchID, name, phone, email, roomType, hashedPass, isActive, briefingViewed).Scan(&id)
	must(err, "insert participant "+name)
	return id
}

var invoiceSeq = 0

func insertInvoice(ctx context.Context, db *sql.DB, participantID, batchID string, amount float64,
	dueDate time.Time, status, issuedBy string, confirmedBy *string, confirmed bool) string {
	invoiceSeq++
	invNo := fmt.Sprintf("INV-%s-%04d", time.Now().Format("200601"), invoiceSeq)
	var id string
	var confirmedAt interface{} = nil
	if confirmed {
		confirmedAt = time.Now().Add(-2 * 24 * time.Hour)
	}
	err := db.QueryRowContext(ctx, `
		INSERT INTO invoices (id, invoice_number, participant_id, batch_id, amount, due_date,
			status, issued_by, confirmed_by, confirmed_at, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id`,
		invNo, participantID, batchID, amount, dueDate, status, issuedBy, confirmedBy, confirmedAt).Scan(&id)
	must(err, "insert invoice")
	return id
}

func insertPaymentProof(ctx context.Context, db *sql.DB, invoiceID string, amount float64, notes,
	status string, reviewedBy *string, reviewNotes string) {
	var reviewedAt interface{} = nil
	if reviewedBy != nil {
		reviewedAt = time.Now().Add(-1 * 24 * time.Hour)
	}
	_, _ = db.ExecContext(ctx, `
		INSERT INTO payment_proofs (id, invoice_id, file_path, amount_claimed, notes, status,
			reviewed_by, review_notes, uploaded_at, reviewed_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW(), $8)`,
		invoiceID, "https://example.com/bukti-transfer-placeholder.jpg",
		amount, notes, status, reviewedBy, reviewNotes, reviewedAt)
}

func insertDocument(ctx context.Context, db *sql.DB, participantID, docType, fileName,
	status, rejectionReason string, reviewedBy *string) {
	var reviewedAt interface{} = nil
	if reviewedBy != nil {
		reviewedAt = time.Now().Add(-1 * 24 * time.Hour)
	}
	_, _ = db.ExecContext(ctx, `
		INSERT INTO documents (id, participant_id, document_type, file_path, file_name,
			status, rejection_reason, reviewed_by, uploaded_at, reviewed_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NULLIF($6,''), $7, NOW(), $8)`,
		participantID, docType, "https://example.com/docs/"+fileName, fileName,
		status, rejectionReason, reviewedBy, reviewedAt)
}

func insertAirportChecklist(ctx context.Context, db *sql.DB, participantID, batchID string,
	baggage, ticket, passport bool, handledBy *string) {
	var bAt, tAt, pAt interface{} = nil, nil, nil
	if baggage {
		bAt = time.Now()
	}
	if ticket {
		tAt = time.Now()
	}
	if passport {
		pAt = time.Now()
	}
	_, _ = db.ExecContext(ctx, `
		INSERT INTO airport_checklists (id, participant_id, batch_id, baggage_checked, baggage_checked_at,
			ticket_distributed, ticket_distributed_at, passport_returned, passport_returned_at,
			handled_by, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (participant_id, batch_id) DO NOTHING`,
		participantID, batchID, baggage, bAt, ticket, tAt, passport, pAt, handledBy)
}

func insertWANotif(ctx context.Context, db *sql.DB, phone, name, msgType, content, refType, refID, status string) {
	var sentAt interface{} = nil
	if status == "sent" {
		sentAt = time.Now()
	}
	_, _ = db.ExecContext(ctx, `
		INSERT INTO wa_notifications (id, recipient_phone, recipient_name, message_type, message_content,
			reference_id, reference_type, status, sent_at, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		phone, name, msgType, content, refID, refType, status, sentAt)
}

func printSummary(ctx context.Context, db *sql.DB) {
	tables := []string{
		"users", "tour_leaders", "packages", "package_images", "package_batches",
		"leads", "lead_notes", "participants", "invoices", "payment_proofs",
		"documents", "country_document_requirements", "airport_checklists", "wa_notifications",
	}
	fmt.Println("\n📊 Data summary:")
	for _, t := range tables {
		var n int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+t).Scan(&n)
		fmt.Printf("   %-32s : %d rows\n", t, n)
	}
}

func bcryptHash(plain string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), 12)
	must(err, "bcrypt")
	return string(h)
}

func daysFromNow(d int) time.Time {
	return time.Now().AddDate(0, 0, d)
}

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("%s: %v", ctx, err)
	}
}
