package service

import (
	"fmt"
	"net/url"
	"strings"
)

const consultantPhone = "6282121952655" // override via env in production

// WhatsAppLinkParams holds the data needed to build a WA consultation link.
type WhatsAppLinkParams struct {
	FullName      string
	Destination   string
	NumPeople     int
	DurationDays  int
	DepartureDate string
	Budget        float64
	Notes         string
	PackageTitle  string
}

// BuildWhatsAppLink generates a pre-filled WhatsApp chat link.
func BuildWhatsAppLink(phone string, p WhatsAppLinkParams) string {
	if phone == "" {
		phone = consultantPhone
	}

	lines := []string{
		"Halo, saya ingin konsultasi paket wisata:",
		"",
		fmt.Sprintf("• Nama: %s", p.FullName),
	}

	if p.PackageTitle != "" {
		lines = append(lines, fmt.Sprintf("• Paket: %s", p.PackageTitle))
	}
	if p.Destination != "" {
		lines = append(lines, fmt.Sprintf("• Destinasi: %s", p.Destination))
	}
	if p.NumPeople > 0 {
		lines = append(lines, fmt.Sprintf("• Jumlah orang: %d", p.NumPeople))
	}
	if p.DurationDays > 0 {
		lines = append(lines, fmt.Sprintf("• Durasi: %d hari", p.DurationDays))
	}
	if p.DepartureDate != "" {
		lines = append(lines, fmt.Sprintf("• Tanggal keberangkatan: %s", p.DepartureDate))
	}
	if p.Budget > 0 {
		lines = append(lines, fmt.Sprintf("• Budget: Rp %.0f", p.Budget))
	}
	if p.Notes != "" {
		lines = append(lines, fmt.Sprintf("• Catatan: %s", p.Notes))
	}

	message := strings.Join(lines, "\n")
	return fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(message))
}
