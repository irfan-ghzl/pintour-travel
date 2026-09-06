package service

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// The rupiah formatter and the title-caser moved to internal/format, tested
// once there for every caller instead of once per package that copied them.

// Every character these templates actually contain has a cp1252 byte, and the
// document's translator produces it. The bullets in the briefing's packing
// list, the mid-dot in its header, the copyright in both footers, and the em
// dash the airport report prints for a step nobody has done — each used to
// reach the page as two or three replacement glyphs.
func TestDocTranslatesToCoreFontEncoding(t *testing.T) {
	d := newDoc("P", 20, 20, 20)
	cases := map[string]byte{
		"•": 0x95, // briefing packing list
		"·": 0xb7, // briefing header separator
		"©": 0xa9, // footers
		"—": 0x97, // airport report, step not done
		"ï": 0xef, // accented participant name
		"é": 0xe9,
	}
	for in, want := range cases {
		got := d.tr(in)
		if len(got) != 1 || got[0] != want {
			t.Errorf("tr(%q) = % x, want %02x", in, got, want)
		}
	}
}

// The writer applies the translation, so text reaches the page encoded whether
// or not whoever wrote the line remembered to ask.
func TestDocWriterTranslatesText(t *testing.T) {
	d := newDoc("P", 20, 20, 20)
	d.SetCompression(false) // so the page's text is readable in the output
	d.SetFont("Helvetica", "", 10)
	d.Cell(0, 6, "Rafi Rahmadhanï")
	d.CellFormat(0, 6, "Total • Rp 1", "", 1, "L", false, 0, "")
	d.MultiCell(0, 6, "© Pintour", "", "L", false)

	var buf bytes.Buffer
	if err := d.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	for _, raw := range []string{"ï", "•", "©"} {
		if bytes.Contains(buf.Bytes(), []byte(raw)) {
			t.Errorf("dokumen memuat %q sebagai UTF-8 mentah — penulis tidak menerjemahkan", raw)
		}
	}
	for _, want := range []byte{0xef, 0x95, 0xa9} {
		if !bytes.Contains(buf.Bytes(), []byte{want}) {
			t.Errorf("dokumen tidak memuat byte cp1252 %02x", want)
		}
	}
}

func TestGenerateInvoice(t *testing.T) {
	svc := NewPDFService()
	data := InvoiceData{
		InvoiceNumber:    "INV-202605-0001",
		IssuedAt:         time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
		DueDate:          calendar.New(2026, time.June, 15),
		ParticipantName:  "Budi Hartono",
		ParticipantPhone: "628123456789",
		PackageName:      "Umroh Plus Istanbul 12 Hari",
		BatchDate:        "15 Juni 2026",
		RoomType:         "double",
		Amount:           25000000,
		Notes:            "Diskon early bird 5%",
		IssuedByName:     "Admin Pintour",
	}
	out, err := svc.GenerateInvoice(data)
	if err != nil {
		t.Fatalf("GenerateInvoice error: %v", err)
	}
	if len(out) < 1000 {
		t.Errorf("PDF too small: %d bytes", len(out))
	}
	// PDFs start with %PDF-
	if !strings.HasPrefix(string(out[:5]), "%PDF-") {
		t.Errorf("output is not a valid PDF (missing %%PDF- header)")
	}
}

func TestGenerateBriefing(t *testing.T) {
	svc := NewPDFService()
	out, err := svc.GenerateBriefing(BriefingData{
		ParticipantName: "Siti Aminah",
		PackageName:     "Umroh Plus Istanbul",
		DepartureDate:   "15 Juni 2026",
		TourLeaderName:  "Ahmad Rizki",
		TourLeaderPhone: "628111222333",
		TourLeaderBio:   "Tour leader berpengalaman 10 tahun",
	})
	if err != nil {
		t.Fatalf("GenerateBriefing error: %v", err)
	}
	if len(out) < 500 {
		t.Errorf("briefing PDF too small: %d bytes", len(out))
	}
}

func TestGenerateAirportReport(t *testing.T) {
	svc := NewPDFService()
	doneAt := time.Date(2026, 6, 15, 4, 30, 0, 0, time.UTC)
	data := AirportReportData{
		BatchName:      "Batch Umroh Juni",
		DepartureDate:  "15 Jun 2026",
		TourLeaderName: "Ahmad Rizki",
		TotalPax:       20,
		DoneCount:      20,
		PendingCount:   0,
		GeneratedAt:    time.Now(),
		Rows: []AirportRow{
			{ParticipantName: "Budi Hartono", BaggageAt: "04:00", TicketAt: "04:15", PassportAt: "04:30"},
			{ParticipantName: "Siti Aminah", BaggageAt: "04:05", TicketAt: "04:20", PassportAt: doneAt.Format("15:04")},
		},
	}
	out, err := svc.GenerateAirportReport(data)
	if err != nil {
		t.Fatalf("GenerateAirportReport error: %v", err)
	}
	if len(out) < 500 {
		t.Errorf("airport report PDF too small: %d bytes", len(out))
	}
}

// Invoice dan briefing memakai satu halaman penuh dengan pita hijau di kaki
// halaman. Pita itu duduk di y=277 dan teksnya di y=281 — di dalam zona yang
// dianggap gofpdf sebagai batas bawah, sehingga pemutus halaman otomatis
// melemparkan teksnya ke halaman berikutnya. Akibatnya invoice tiba dengan
// lembar kedua yang tampak kosong: huruf putih di atas kertas putih.
//
// Jumlah halaman dibaca dari berkasnya sendiri, bukan dari keadaan internal
// gofpdf, karena yang diterima peserta adalah berkas itu.
func countPDFPages(t *testing.T, b []byte) int {
	t.Helper()
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("bukan berkas PDF: %q", string(b[:min(8, len(b))]))
	}
	return bytes.Count(b, []byte("/Type /Page")) - bytes.Count(b, []byte("/Type /Pages"))
}

func TestGenerateInvoice_FitsOnOnePage(t *testing.T) {
	out, err := (&PDFService{}).GenerateInvoice(InvoiceData{
		InvoiceNumber: "INV-202608-0010", IssuedAt: time.Now(),
		DueDate:         calendar.Today().AddDays(7),
		ParticipantName: "agiel", ParticipantPhone: "62895334442331",
		PackageName: "Korea Selatan Honeymoon 7 Hari", BatchDate: "09 September 2026",
		RoomType: "Double", Amount: 189000000,
		Notes: "Invoice otomatis dibuat saat konversi leads.", IssuedByName: "Admin Pintour",
	})
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	if n := countPDFPages(t, out); n != 1 {
		t.Fatalf("invoice = %d halaman, ingin 1", n)
	}
}

func TestGenerateBriefing_FitsOnOnePage(t *testing.T) {
	out, err := (&PDFService{}).GenerateBriefing(BriefingData{
		ParticipantName: "agiel", PackageName: "Korea Selatan Honeymoon 7 Hari",
		DepartureDate: "09 September 2026", TourLeaderName: "Siti",
		TourLeaderPhone: "628111000001", TourLeaderBio: "Tour leader berpengalaman.",
	})
	if err != nil {
		t.Fatalf("GenerateBriefing: %v", err)
	}
	if n := countPDFPages(t, out); n != 1 {
		t.Fatalf("briefing = %d halaman, ingin 1", n)
	}
}

// A4 selebar 210 mm dengan margin 20 mm di kiri dan kanan menyisakan 170 mm.
// Setiap kelompok kolom harus berjumlah persis itu.
//
// Ketiga tabel dulu melebihinya — invoice 190 mm pada tiga barisnya, laporan
// bandara 180 mm — dan tidak ada yang mengeluh: gofpdf menggambar terus melewati
// margin. Yang terlihat oleh peserta hanyalah tabel yang menempel di tepi
// kertas sementara garis pemisah di atasnya berhenti di tempat yang benar.
func TestPDFColumnsFitTheContentWidth(t *testing.T) {
	cases := map[string][]float64{
		"baris tanggal invoice": {contentWidth / 2, contentWidth / 2},
		"tabel invoice":         {invNo, invDesc, invValue},
		"bilah total invoice":   {contentWidth - invValue, invValue},
		"tabel laporan bandara": {airNo, airName, airStep, airStep, airLast},
	}
	for nama, kolom := range cases {
		var total float64
		for _, w := range kolom {
			if w <= 0 {
				t.Fatalf("%s: ada kolom selebar %v", nama, w)
			}
			total += w
		}
		if total != contentWidth {
			t.Errorf("%s: jumlah lebar = %v mm, ingin %v mm", nama, total, contentWidth)
		}
	}
}
