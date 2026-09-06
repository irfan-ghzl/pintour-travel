package service

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	"github.com/irfan-ghzl/pintour-travel/internal/format"
)

// PDFService generates PDF documents.
type PDFService struct{}

func NewPDFService() *PDFService { return &PDFService{} }

// doc is a gofpdf document that translates text on the way in.
//
// The core fonts these documents use (Helvetica and friends) are not Unicode:
// they take cp1252, so a Go string containing "·", "•", "©" or an em dash
// reaches the page as two or three replacement glyphs. Every one of those
// appears in these templates — the briefing's bullet lists, the invoice's
// footer, the airport report's "—" for a step nobody has done yet — and every
// one of them was printing as mojibake.
//
// Wrapping the writer rather than sprinkling tr() over sixty call sites means
// the next line of text added here is translated because it is written, not
// because someone remembered. The report module already did this correctly and
// is the pattern being followed.
type doc struct {
	*gofpdf.Fpdf
	tr func(string) string
}

func newDoc(orientation string, marginLeft, marginTop, marginRight float64) *doc {
	f := gofpdf.New(orientation, "mm", "A4", "")
	f.SetMargins(marginLeft, marginTop, marginRight)
	d := &doc{Fpdf: f, tr: f.UnicodeTranslatorFromDescriptor("")}
	f.AddPage()
	return d
}

func (d *doc) Cell(w, h float64, txt string) { d.Fpdf.Cell(w, h, d.tr(txt)) }

// contentWidth is how much room a row actually has: A4 is 210 mm wide and every
// document here is built with 20 mm margins on both sides.
//
// It exists because the number was never written down, so each table was sized
// by eye and every one of them overflowed — the invoice by 20 mm, the airport
// report by 10. Nothing complained: gofpdf draws happily past the margin, and
// the result only shows up as a table whose right edge is flush against the
// paper while the divider above it stops where it should.
const contentWidth = 170.0

// Kolom tabel invoice dan laporan bandara. Masing-masing kelompok dijumlahkan
// menjadi contentWidth, dan TestPDFColumnsFitTheContentWidth menjaganya tetap
// begitu — angka yang dipilih dengan mata adalah cara cacat ini muncul.
const (
	invNo    = 10.0
	invDesc  = 90.0
	invValue = contentWidth - invNo - invDesc

	airNo    = 8.0
	airName  = 64.0
	airStep  = 32.0
	airLast  = contentWidth - airNo - airName - 2*airStep
)

// footerBand draws the closing green band across the bottom of the page.
//
// Auto page break is switched off first, and that is the whole point. gofpdf
// breaks when a cell would end past 2 cm from the bottom, while this band sits
// at y=277 with its text at y=281 — deliberately inside that zone, because it is
// the bottom of the page. Left on, the band was painted on page one but its text
// pushed onto a fresh page two: white letters on white paper, so the invoice
// arrived with a blank second sheet that nobody could see anything on.
func (d *doc) footerBand(text string) {
	d.SetAutoPageBreak(false, 0)
	d.SetFillColor(16, 100, 59)
	d.Rect(0, 277, 210, 20, "F")
	d.SetTextColor(255, 255, 255)
	d.SetFont("Helvetica", "", 8)
	d.SetXY(20, 281)
	d.Cell(0, 5, text)
}

func (d *doc) CellFormat(w, h float64, txt, border string, ln int, align string, fill bool, link int, linkStr string) {
	d.Fpdf.CellFormat(w, h, d.tr(txt), border, ln, align, fill, link, linkStr)
}

func (d *doc) MultiCell(w, h float64, txt, border, align string, fill bool) {
	d.Fpdf.MultiCell(w, h, d.tr(txt), border, align, fill)
}

// InvoiceData holds all data needed to render an invoice PDF.
type InvoiceData struct {
	InvoiceNumber    string
	IssuedAt         time.Time
	DueDate          calendar.Date
	ParticipantName  string
	ParticipantPhone string
	PackageName      string
	BatchDate        string
	RoomType         string
	Amount           float64
	Notes            string
	IssuedByName     string
}

// GenerateInvoice creates an invoice PDF and returns the bytes.
func (s *PDFService) GenerateInvoice(d InvoiceData) ([]byte, error) {
	pdf := newDoc("P", 20, 20, 20)

	// Header
	pdf.SetFillColor(16, 100, 59) // emerald-700
	pdf.Rect(0, 0, 210, 40, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(20, 10)
	pdf.Cell(0, 10, "PINTOUR TRAVEL")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(20, 22)
	pdf.Cell(0, 8, "Agen Perjalanan Wisata & Umroh Terpercaya")

	// Invoice badge
	pdf.SetFillColor(255, 255, 255)
	pdf.RoundedRect(140, 10, 50, 20, 3, "1234", "F")
	pdf.SetTextColor(16, 100, 59)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(143, 14)
	pdf.Cell(44, 5, "INVOICE")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(143, 20)
	pdf.Cell(44, 6, d.InvoiceNumber)

	// Reset colors
	pdf.SetTextColor(30, 30, 30)
	pdf.SetY(50)

	// Invoice meta
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(contentWidth/2, 5, fmt.Sprintf("Tanggal Terbit: %s", d.IssuedAt.Format("02 January 2006")), "", 0, "L", false, 0, "")
	pdf.CellFormat(contentWidth/2, 5, fmt.Sprintf("Jatuh Tempo: %s", d.DueDate.Format("02 January 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Divider
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(6)

	// Tagihan kepada
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(30, 30, 30)
	pdf.Cell(0, 6, "Tagihan Kepada:")
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(0, 7, d.ParticipantName)
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(0, 5, fmt.Sprintf("WhatsApp: %s", d.ParticipantPhone))
	pdf.Ln(10)

	// Detail tabel
	pdf.SetFillColor(240, 253, 244)
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(invNo, 8, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(invDesc, 8, "Keterangan", "1", 0, "L", true, 0, "")
	pdf.CellFormat(invValue, 8, "Nilai", "1", 1, "R", true, 0, "")

	items := []struct{ desc, val string }{
		{"Paket Wisata", d.PackageName},
		{"Tanggal Keberangkatan", d.BatchDate},
		{"Tipe Kamar", format.Title(d.RoomType)},
	}

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetFillColor(255, 255, 255)
	for i, item := range items {
		fill := i%2 == 1
		if fill {
			pdf.SetFillColor(248, 248, 248)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(invNo, 7, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(invDesc, 7, item.desc, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(invValue, 7, item.val, "1", 1, "R", fill, 0, "")
	}
	pdf.Ln(4)

	// Total
	pdf.SetFillColor(16, 100, 59)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(contentWidth-invValue, 10, "TOTAL TAGIHAN", "0", 0, "R", true, 0, "")
	pdf.CellFormat(invValue, 10, fmt.Sprintf("Rp %s", format.Rupiah(d.Amount)), "0", 1, "R", true, 0, "")
	pdf.SetTextColor(30, 30, 30)
	pdf.Ln(8)

	// Notes
	if d.Notes != "" {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.SetTextColor(100, 100, 100)
		pdf.Cell(0, 5, fmt.Sprintf("Catatan: %s", d.Notes))
		pdf.Ln(8)
	}

	// Payment info
	pdf.SetFillColor(254, 252, 232)
	pdf.SetDrawColor(202, 138, 4)
	pdf.RoundedRect(20, pdf.GetY(), 170, 30, 3, "1234", "FD")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(120, 70, 0)
	pdf.SetXY(25, pdf.GetY()+4)
	pdf.Cell(0, 6, "Cara Pembayaran")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 80, 0)
	pdf.SetXY(25, pdf.GetY()+6)
	pdf.Cell(0, 5, "Transfer ke rekening yang diberikan tim Pintour, kemudian upload bukti transfer")
	pdf.SetXY(25, pdf.GetY()+5)
	pdf.Cell(0, 5, "melalui Portal Peserta. Admin akan mengkonfirmasi dalam 1x24 jam.")
	pdf.Ln(35)

	pdf.footerBand(fmt.Sprintf("Diterbitkan oleh: %s  |  %s  |  pintour.app",
		d.IssuedByName, d.IssuedAt.Format("02 Jan 2006 15:04")))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generate invoice PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// BriefingData holds data for briefing PDF generation.
type BriefingData struct {
	ParticipantName string
	PackageName     string
	DepartureDate   string
	TourLeaderName  string
	TourLeaderPhone string
	TourLeaderBio   string
}

// GenerateBriefing creates a briefing PDF.
func (s *PDFService) GenerateBriefing(d BriefingData) ([]byte, error) {
	pdf := newDoc("P", 20, 20, 20)

	// Header
	pdf.SetFillColor(16, 100, 59)
	pdf.Rect(0, 0, 210, 35, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(20, 8)
	pdf.Cell(0, 10, "MATERI BRIEFING PERJALANAN")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(20, 22)
	pdf.Cell(0, 7, fmt.Sprintf("%s  ·  Keberangkatan %s", d.PackageName, d.DepartureDate))

	pdf.SetTextColor(30, 30, 30)
	pdf.SetY(45)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(0, 5, fmt.Sprintf("Kepada Yth. Bapak/Ibu %s", d.ParticipantName))
	pdf.Ln(10)

	sections := []struct{ title, body string }{
		{"Tour Leader Anda", fmt.Sprintf("Nama: %s\nKontak Darurat: %s\n%s", d.TourLeaderName, d.TourLeaderPhone, d.TourLeaderBio)},
		{"Tata Tertib Perjalanan", "1. Hadir di titik kumpul 3 jam sebelum jadwal penerbangan\n2. Bawa paspor asli dan semua dokumen perjalanan\n3. Batasan bagasi: 20kg check-in + 7kg kabin\n4. Patuhi jadwal dan arahan tour leader\n5. Hormati budaya dan adat setempat\n6. Tidak meninggalkan grup tanpa izin tour leader"},
		{"Barang Bawaan Disarankan", "• Pakaian sesuai cuaca tujuan\n• Obat-obatan pribadi\n• Kartu debit/kredit internasional\n• Adaptor universal & powerbank\n• Uang tunai secukupnya\n• Dokumen perjalanan (fotokopi)\n• Kamera/HP + charger"},
		{"Panduan Keimigrasian", "• Antri di jalur WNA saat tiba di luar negeri\n• Isi formulir imigrasi sebelum mendarat\n• Ikuti arahan tour leader untuk custom clearance\n• Simpan boarding pass hingga tiba di tujuan"},
		{"Kontak Darurat", fmt.Sprintf("Tour Leader: %s (%s)\nKantor Pintour: +62 (hubungi admin)\nHotline 24 jam tersedia selama perjalanan", d.TourLeaderName, d.TourLeaderPhone)},
	}

	for _, sec := range sections {
		pdf.SetFillColor(240, 253, 244)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(16, 100, 59)
		pdf.CellFormat(contentWidth, 7, "  "+sec.title, "LTR", 1, "L", true, 0, "")
		pdf.SetFillColor(250, 250, 250)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(50, 50, 50)
		pdf.MultiCell(170, 5, sec.body, "LBR", "L", true)
		pdf.Ln(4)
	}

	pdf.footerBand("Dokumen ini bersifat rahasia dan hanya untuk peserta terdaftar. © Pintour Travel")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generate briefing PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// AirportReportData holds data for airport handling post-report (FR-AIR-06).
type AirportReportData struct {
	BatchName      string
	DepartureDate  string
	TourLeaderName string
	TotalPax       int
	DoneCount      int
	PendingCount   int
	StartedAt      *time.Time // jam handling mulai (timestamp checklist pertama)
	FinishedAt     *time.Time // jam handling selesai (timestamp checklist terakhir)
	GeneratedAt    time.Time
	Rows           []AirportRow
}

type AirportRow struct {
	ParticipantName string
	BaggageAt       string
	TicketAt        string
	PassportAt      string
}

// GenerateAirportReport creates post-handling report PDF (FR-AIR-06).
func (s *PDFService) GenerateAirportReport(d AirportReportData) ([]byte, error) {
	pdf := newDoc("P", 15, 15, 15)

	// Header
	pdf.SetFillColor(16, 100, 59)
	pdf.Rect(0, 0, 210, 35, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(15, 8)
	pdf.Cell(0, 10, "LAPORAN AIRPORT HANDLING")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(15, 22)
	pdf.Cell(0, 6, fmt.Sprintf("Batch: %s · %s", d.BatchName, d.DepartureDate))

	pdf.SetTextColor(30, 30, 30)
	pdf.SetY(45)

	// Summary
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 6, "Ringkasan")
	pdf.Ln(7)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(50, 5, "Total Peserta")
	pdf.Cell(0, 5, fmt.Sprintf(": %d orang", d.TotalPax))
	pdf.Ln(5)
	pdf.Cell(50, 5, "Selesai Diproses")
	pdf.Cell(0, 5, fmt.Sprintf(": %d orang", d.DoneCount))
	pdf.Ln(5)
	pdf.Cell(50, 5, "Menunggu")
	pdf.Cell(0, 5, fmt.Sprintf(": %d orang", d.PendingCount))
	pdf.Ln(5)
	pdf.Cell(50, 5, "Tour Leader Bertugas")
	pdf.Cell(0, 5, fmt.Sprintf(": %s", d.TourLeaderName))
	pdf.Ln(5)
	// AC-AIR-02: jam mulai & selesai handling
	startedStr := "—"
	if d.StartedAt != nil {
		startedStr = d.StartedAt.Format("15:04")
	}
	finishedStr := "—"
	if d.FinishedAt != nil {
		finishedStr = d.FinishedAt.Format("15:04")
	}
	pdf.Cell(50, 5, "Mulai Handling")
	pdf.Cell(0, 5, fmt.Sprintf(": %s", startedStr))
	pdf.Ln(5)
	pdf.Cell(50, 5, "Selesai Handling")
	pdf.Cell(0, 5, fmt.Sprintf(": %s", finishedStr))
	pdf.Ln(5)
	pdf.Cell(50, 5, "Laporan Dibuat")
	pdf.Cell(0, 5, fmt.Sprintf(": %s", d.GeneratedAt.Format("02 Jan 2006 15:04")))
	pdf.Ln(10)

	// Table header
	pdf.SetFillColor(240, 253, 244)
	pdf.SetTextColor(30, 30, 30)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(airNo, 7, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(airName, 7, "Nama Peserta", "1", 0, "L", true, 0, "")
	pdf.CellFormat(airStep, 7, "Bagasi", "1", 0, "C", true, 0, "")
	pdf.CellFormat(airStep, 7, "Tiket", "1", 0, "C", true, 0, "")
	pdf.CellFormat(airLast, 7, "Paspor", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	for i, row := range d.Rows {
		fill := i%2 == 1
		if fill {
			pdf.SetFillColor(248, 248, 248)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(airNo, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(airName, 6, row.ParticipantName, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(airStep, 6, row.BaggageAt, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(airStep, 6, row.TicketAt, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(airLast, 6, row.PassportAt, "1", 1, "C", fill, 0, "")
	}

	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.Cell(0, 5, "Laporan ini dihasilkan otomatis oleh sistem Pintour. © Pintour Travel")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generate airport report PDF: %w", err)
	}
	return buf.Bytes(), nil
}
