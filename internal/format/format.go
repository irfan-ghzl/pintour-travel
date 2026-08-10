// Package format holds the presentation helpers more than one layer needs.
//
// Each of these existed several times over before it lived here: the rupiah
// separator in four packages, text truncation in two — one of which cut on byte
// boundaries and could split a character in half. They are pure functions over
// their arguments, which is why a shared package costs nothing: no state, no
// configuration, and no reason for two callers to want different answers.
package format

import (
	"fmt"
	"strings"
)

// Rupiah renders an amount with Indonesian thousands separators and no symbol:
// 5000000 becomes "5.000.000". Fractions are dropped — invoices are issued in
// whole rupiah, and a stray ".5" in a total reads as a bug to whoever sees it.
func Rupiah(amount float64) string {
	digits := fmt.Sprintf("%.0f", amount)
	n := len(digits)
	out := make([]byte, 0, n+n/3)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digits[i])
	}
	return string(out)
}

// TruncateRunes shortens s to at most max characters, cutting between
// characters rather than inside one.
//
// Counting bytes is what makes this worth a function: "Rafi Nur Rahmadhanï" is
// nineteen characters and twenty bytes, and a byte-wise cut at nineteen leaves
// half of the ï behind — which every reader downstream then renders as a
// replacement box.
func TruncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// Ellipsis is TruncateRunes with a trailing "…", so a shortened value looks
// shortened. The ellipsis counts toward max.
func Ellipsis(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len([]rune(s)) <= max {
		return s
	}
	return TruncateRunes(s, max-1) + "…"
}

// Title upper-cases the first letter of s and leaves the rest alone, for the
// single-word status labels the PDFs print. It is rune-aware, unlike the
// byte-arithmetic version it replaces, which turned any non-ASCII first letter
// into a different character entirely.
func Title(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}
