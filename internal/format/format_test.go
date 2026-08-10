package format

import "testing"

func TestRupiah(t *testing.T) {
	cases := map[float64]string{
		0:          "0",
		5:          "5",
		999:        "999",
		1000:       "1.000",
		5_000_000:  "5.000.000",
		12_345_678: "12.345.678",
		1500.4:     "1.500",
		1500.6:     "1.501", // %.0f rounds
		-5000:      "-5.000",
	}
	for in, want := range cases {
		if got := Rupiah(in); got != want {
			t.Errorf("Rupiah(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	type in struct {
		s   string
		max int
	}
	cases := map[in]string{
		{"Budi Santoso", 40}:    "Budi Santoso",
		{"Budi Santoso", 12}:    "Budi Santoso",
		{"Budi Santoso", 4}:     "Budi",
		{"Budi Santoso", 0}:     "",
		{"Budi Santoso", -1}:    "",
		{"", 10}:                "",
		{"Rafi Rahmadhanï", 15}: "Rafi Rahmadhanï",
		// The byte-wise version cut here and left half of the ï behind.
		{"Rafi Rahmadhanï", 14}: "Rafi Rahmadhan",
		{"café", 4}:             "café",
		{"café", 3}:             "caf",
		{"日本語テキスト", 3}:          "日本語",
	}
	for tc, want := range cases {
		if got := TruncateRunes(tc.s, tc.max); got != want {
			t.Errorf("TruncateRunes(%q, %d) = %q, want %q", tc.s, tc.max, got, want)
		}
	}
}

func TestEllipsis(t *testing.T) {
	type in struct {
		s   string
		max int
	}
	cases := map[in]string{
		{"Budi Santoso", 40}: "Budi Santoso",
		{"Budi Santoso", 12}: "Budi Santoso",
		{"Budi Santoso", 6}:  "Budi …",
		{"Budi Santoso", 1}:  "…",
		{"Budi Santoso", 0}:  "",
		{"日本語テキスト", 4}:       "日本語…",
	}
	for tc, want := range cases {
		if got := Ellipsis(tc.s, tc.max); got != want {
			t.Errorf("Ellipsis(%q, %d) = %q, want %q", tc.s, tc.max, got, want)
		}
	}
}

// The result never exceeds max characters — the property the report's column
// widths depend on.
func TestEllipsisRespectsWidth(t *testing.T) {
	for _, s := range []string{"", "a", "abcdefghij", "日本語テキストです", "Rafi Rahmadhanï panjang sekali namanya"} {
		for max := 1; max <= 12; max++ {
			if n := len([]rune(Ellipsis(s, max))); n > max {
				t.Errorf("Ellipsis(%q, %d) menghasilkan %d karakter", s, max, n)
			}
		}
	}
}

func TestTitle(t *testing.T) {
	cases := map[string]string{
		"tersedia": "Tersedia",
		"penuh":    "Penuh",
		"Ditutup":  "Ditutup",
		"":         "",
		"a":        "A",
		"élan":     "Élan",
	}
	for in, want := range cases {
		if got := Title(in); got != want {
			t.Errorf("Title(%q) = %q, want %q", in, got, want)
		}
	}
}
