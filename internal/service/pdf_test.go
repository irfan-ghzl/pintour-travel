package service

import (
	"strings"
	"testing"
	"time"
)

func TestFormatRupiah(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1.000"},
		{1500000, "1.500.000"},
		{25000000, "25.000.000"},
		{1234567890, "1.234.567.890"},
	}
	for _, c := range cases {
		got := formatRupiah(c.in)
		if got != c.want {
			t.Errorf("formatRupiah(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTitelize(t *testing.T) {
	cases := map[string]string{
		"single": "Single",
		"double": "Double",
		"triple": "Triple",
		"":       "",
	}
	for in, want := range cases {
		if got := titelize(in); got != want {
			t.Errorf("titelize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenerateInvoice(t *testing.T) {
	svc := NewPDFService()
	data := InvoiceData{
		InvoiceNumber:    "INV-202605-0001",
		IssuedAt:         time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
		DueDate:          time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
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
