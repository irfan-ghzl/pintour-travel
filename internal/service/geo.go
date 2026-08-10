package service

import "strings"

// DestinationToCountryCode maps a free-text destination (e.g. "Jepang",
// "Arab Saudi & Turki") to the 2-letter ISO country code used in the
// country_document_requirements table. Substring match only; for multi-country
// destinations the first matching country wins. Falls back to "ID".
func DestinationToCountryCode(destination string) string {
	d := strings.ToLower(destination)
	mapping := []struct {
		needle string
		code   string
	}{
		{"jepang", "JP"}, {"japan", "JP"},
		{"korea", "KR"},
		{"turki", "TR"}, {"turkey", "TR"},
		{"arab saudi", "SA"}, {"saudi", "SA"}, {"makkah", "SA"}, {"madinah", "SA"},
		{"uea", "AE"}, {"emirat", "AE"}, {"dubai", "AE"}, {"abu dhabi", "AE"},
		{"singapore", "SG"}, {"singapura", "SG"},
		{"malaysia", "MY"},
		{"thailand", "TH"},
		{"vietnam", "VN"},
		{"china", "CN"}, {"tiongkok", "CN"},
		{"hong kong", "HK"}, {"hongkong", "HK"},
		{"taiwan", "TW"},
		{"australia", "AU"},
		{"belanda", "NL"}, {"netherlands", "NL"},
		{"perancis", "FR"}, {"prancis", "FR"}, {"france", "FR"},
		{"jerman", "DE"}, {"germany", "DE"},
		{"italia", "IT"}, {"italy", "IT"},
		{"spanyol", "ES"}, {"spain", "ES"},
		{"swiss", "CH"},
		{"inggris", "GB"}, {"uk", "GB"},
		{"amerika", "US"}, {"usa", "US"},
		{"kanada", "CA"}, {"canada", "CA"},
		{"mesir", "EG"}, {"egypt", "EG"},
		{"yordania", "JO"}, {"jordan", "JO"},
		{"indonesia", "ID"},
	}
	for _, m := range mapping {
		if strings.Contains(d, m.needle) {
			return m.code
		}
	}
	return "ID"
}
