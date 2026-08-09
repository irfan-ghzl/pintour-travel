// TODO(ocr-v2.0-F3): unit test untuk OCR parser/validator — tetap valid, aktifkan bersama ocr.go.
package service

import (
	"testing"
	"time"
)

func TestParseDocumentText_KTP(t *testing.T) {
	text := `PROVINSI DKI JAKARTA
NIK : 3174012501900001
Nama : JOHN DOE
Tempat/Tgl Lahir : JAKARTA, 25-01-1990`

	r := ParseDocumentText(text, "ktp")
	if r.DocumentNumber != "3174012501900001" {
		t.Errorf("NIK = %q, want 3174012501900001", r.DocumentNumber)
	}
	if r.Name != "JOHN DOE" {
		t.Errorf("Name = %q, want JOHN DOE", r.Name)
	}
	if r.BirthDate != "1990-01-25" {
		t.Errorf("BirthDate = %q, want 1990-01-25", r.BirthDate)
	}
	if r.Confidence < 0.99 {
		t.Errorf("Confidence = %v, want ~1.0", r.Confidence)
	}
}

func TestParseDocumentText_PassportMRZ(t *testing.T) {
	l1 := "P<IDNDOE<<JOHN<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	l2 := "A1234567<0IDN9001010M3001010<<<<<<<<<<<<<<00"
	r := ParseDocumentText(l1+"\n"+l2, "passport")

	if r.DocumentNumber != "A1234567" {
		t.Errorf("passport no = %q, want A1234567", r.DocumentNumber)
	}
	if r.Nationality != "IDN" {
		t.Errorf("nationality = %q, want IDN", r.Nationality)
	}
	if r.ExpiryDate != "2030-01-01" {
		t.Errorf("expiry = %q, want 2030-01-01", r.ExpiryDate)
	}
	if r.BirthDate != "1990-01-01" {
		t.Errorf("birth = %q, want 1990-01-01", r.BirthDate)
	}
}

func TestValidateOCRResult_KTP(t *testing.T) {
	s := &OCRService{threshold: 0.85}
	ok := s.ValidateOCRResult(&OCRExtraction{DocumentNumber: "3174012501900001", Confidence: 1.0}, "ktp", time.Time{})
	if !ok.Passed {
		t.Errorf("expected valid KTP to pass, got: %s", ok.Notes)
	}
	bad := s.ValidateOCRResult(&OCRExtraction{DocumentNumber: "123", Confidence: 1.0}, "ktp", time.Time{})
	if bad.Passed {
		t.Error("expected short NIK to fail")
	}
}

func TestValidateOCRResult_Passport(t *testing.T) {
	s := &OCRService{threshold: 0.85}
	departure := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)

	valid := s.ValidateOCRResult(&OCRExtraction{ExpiryDate: "2030-01-01", Confidence: 0.9}, "passport", departure)
	if !valid.Passed {
		t.Errorf("expected valid passport to pass, got: %s", valid.Notes)
	}
	expired := s.ValidateOCRResult(&OCRExtraction{ExpiryDate: "2026-08-01", Confidence: 0.9}, "passport", departure)
	if expired.Passed {
		t.Error("expected passport expiring within 6 months of departure to fail")
	}
	lowConf := s.ValidateOCRResult(&OCRExtraction{ExpiryDate: "2030-01-01", Confidence: 0.3}, "passport", departure)
	if lowConf.Passed {
		t.Error("expected low-confidence result to fail")
	}
}

// A scan whose lines carry leading whitespace — which is most of them — reads
// the same as one that does not. The extractor used to match the trimmed line
// but cut the untrimmed one, eating one character of the value per space.
func TestParseDocumentText_KTPWithLeadingWhitespace(t *testing.T) {
	cases := map[string]string{
		"tanpa spasi":  "NIK : 3174012501900001\nNama : JOHN DOE\n",
		"satu spasi":   " NIK : 3174012501900001\n Nama : JOHN DOE\n",
		"dua spasi":    "  NIK : 3174012501900001\n  Nama : JOHN DOE\n",
		"tab":          "\tNIK : 3174012501900001\n\tNama : JOHN DOE\n",
		"spasi campur": "   NIK: 3174012501900001\n \tNama:  JOHN DOE  \n",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			r := ParseDocumentText(text, "ktp")
			if r.Name != "JOHN DOE" {
				t.Errorf("Name = %q, want JOHN DOE", r.Name)
			}
			if r.DocumentNumber != "3174012501900001" {
				t.Errorf("NIK = %q, want 3174012501900001", r.DocumentNumber)
			}
		})
	}
}

func TestFindLabeled(t *testing.T) {
	text := "  Nama : Budi Santoso\nAlamat:  Jl. Merdeka 1\n\tPekerjaan\tGuru\nKosong :   \n"
	cases := map[string]string{
		"Nama":      "Budi Santoso",
		"nama":      "Budi Santoso", // label matching is case-insensitive
		"Alamat":    "Jl. Merdeka 1",
		"Pekerjaan": "Guru",
		"Kosong":    "", // a label with nothing after it yields nothing
		"Golongan":  "", // absent label
	}
	for label, want := range cases {
		if got := findLabeled(text, label); got != want {
			t.Errorf("findLabeled(%q) = %q, want %q", label, got, want)
		}
	}
}
