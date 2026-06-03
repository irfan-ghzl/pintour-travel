package httpdelivery

import "testing"

func TestDestinationToCountryCode(t *testing.T) {
	cases := map[string]string{
		"Jepang":                "JP",
		"Tokyo - Jepang":        "JP",
		"japan":                 "JP",
		"Arab Saudi":            "SA",
		"Umroh - Makkah & Madinah": "SA",
		"Turki - Istanbul":      "TR",
		"Singapore":             "SG",
		"Singapura":             "SG",
		"Malaysia":              "MY",
		"Dubai UAE":             "AE",
		"Korea Selatan":         "KR",
		"Thailand - Bangkok":    "TH",
		"Belanda & Eropa":       "NL",
		"Indonesia":             "ID",
		"Mars":                  "ID", // fallback default
		"":                      "ID",
	}
	for in, want := range cases {
		got := destinationToCountryCode(in)
		if got != want {
			t.Errorf("destinationToCountryCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"08123456789":   "628123456789",
		"+628123456789": "628123456789",
		"628123456789":  "628123456789",
		"":              "",
	}
	for in, want := range cases {
		got := normalizePhone(in)
		if got != want {
			t.Errorf("normalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}
