package httpdelivery

// Ticket 14 — the date contract between the admin forms and the API.
//
// Both forms bind an `<input type="date">` straight into the request body, so
// what they send is a plain "YYYY-MM-DD". The fields those values landed in
// were time.Time, which only decodes RFC3339, so the request failed before it
// reached a handler and the admin saw a generic error with no mention of a
// date. Publishing an invoice and opening a departure batch were both
// impossible from the interface.
//
// The payloads below are copied from the forms — AdminInvoicesPage's
// CreateInvoiceRequest and AdminPackagesPage's batch form — field for field, so
// this contract cannot break again without a test failing.

import (
	"net/http"
	"testing"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// invoiceFormPayload is exactly what AdminInvoicesPage posts.
func invoiceFormPayload(dueDate string) map[string]any {
	return map[string]any{
		"participant_id": "participant-1",
		"batch_id":       "batch-1",
		"amount":         25000000,
		"due_date":       dueDate,
		"notes":          "",
	}
}

// batchFormPayload is exactly what AdminPackagesPage posts, including the
// tour_leader_id it nulls out when nobody is assigned.
func batchFormPayload(departure, returnDate string) map[string]any {
	return map[string]any{
		"departure_date": departure,
		"return_date":    returnDate,
		"quota":          20,
		"price_single":   30000000,
		"price_double":   25000000,
		"price_triple":   22500000,
		"status":         "tersedia",
		"tour_leader_id": nil,
		"wa_group_link":  "",
	}
}

func TestAdminInvoiceForm_AcceptsThePayloadItActuallySends(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.as("admin").POST("/api/v1/admin/invoices", invoiceFormPayload("2026-08-15")).
		expectCode(http.StatusCreated)

	inv := h.Invoices.invoices[h.Invoices.order[len(h.Invoices.order)-1]]
	if got := inv.DueDate.String(); got != "2026-08-15" {
		t.Errorf("due_date tersimpan = %q, want 2026-08-15 — tanggal yang dipilih admin", got)
	}
}

func TestAdminBatchForm_AcceptsThePayloadItActuallySends(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.as("admin").POST("/api/v1/admin/packages/package-1/batches",
		batchFormPayload("2026-09-01", "2026-09-10")).
		expectCode(http.StatusCreated)

	var stored *struct{ departure, ret string }
	for _, b := range h.Batches.batches {
		if b.PackageID == "package-1" && b.DepartureDate.String() == "2026-09-01" {
			stored = &struct{ departure, ret string }{b.DepartureDate.String(), b.ReturnDate.String()}
		}
	}
	if stored == nil {
		t.Fatal("batch tidak tersimpan dengan tanggal berangkat yang dikirim formulir")
	}
	if stored.ret != "2026-09-10" {
		t.Errorf("return_date tersimpan = %q, want 2026-09-10", stored.ret)
	}
}

// The day the admin picked is the day that comes back — no timezone shifts it
// by one in either direction.
func TestAdminDates_SurviveTheRoundTripUnshifted(t *testing.T) {
	// Year boundaries and a mid-year date: the days a one-off shift is most
	// visible on.
	for _, day := range []string{"2026-01-01", "2026-06-30", "2026-08-15", "2026-12-31"} {
		t.Run(day, func(t *testing.T) {
			h := newHarness(t)
			h.seedBaseline()

			h.as("admin").POST("/api/v1/admin/invoices", invoiceFormPayload(day)).
				expectCode(http.StatusCreated)

			res := h.as("admin").GET("/api/v1/admin/invoices")
			res.expectCode(http.StatusOK)
			var envelope struct {
				Data []struct {
					DueDate string `json:"due_date"`
				} `json:"data"`
			}
			res.decode(&envelope)
			var found bool
			for _, inv := range envelope.Data {
				if inv.DueDate == day {
					found = true
				}
			}
			if !found {
				var seen []string
				for _, inv := range envelope.Data {
					seen = append(seen, inv.DueDate)
				}
				t.Errorf("due_date %q tidak ditemukan saat dibaca kembali; yang ada: %v", day, seen)
			}
		})
	}
}

// An older client that still sends RFC3339 keeps working — the API widened,
// it did not swap one exclusive format for another.
func TestAdminInvoiceForm_StillAcceptsRFC3339(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	h.as("admin").POST("/api/v1/admin/invoices", invoiceFormPayload("2026-08-15T00:00:00Z")).
		expectCode(http.StatusCreated)

	inv := h.Invoices.invoices[h.Invoices.order[len(h.Invoices.order)-1]]
	if got := inv.DueDate.String(); got != "2026-08-15" {
		t.Errorf("due_date = %q, want 2026-08-15", got)
	}
}

// A date the API cannot read is refused with a message naming the value, not
// the generic fallback the form used to show for every failure.
func TestAdminDates_RejectionNamesTheDate(t *testing.T) {
	cases := map[string]string{
		"format terbalik":    "15-08-2026",
		"pakai garis miring": "2026/08/15",
		"teks bebas":         "besok",
		"bulan tidak ada":    "2026-13-01",
	}

	for name, bad := range cases {
		t.Run("invoice/"+name, func(t *testing.T) {
			h := newHarness(t)
			h.seedBaseline()
			before := len(h.Invoices.order)
			res := h.as("admin").POST("/api/v1/admin/invoices", invoiceFormPayload(bad))
			res.expectCode(http.StatusBadRequest)
			res.expectMessageMentions("tanggal")
			if len(h.Invoices.order) != before {
				t.Error("invoice tersimpan meski tanggalnya ditolak")
			}
		})

		t.Run("batch/"+name, func(t *testing.T) {
			h := newHarness(t)
			h.seedBaseline()
			res := h.as("admin").POST("/api/v1/admin/packages/package-1/batches",
				batchFormPayload(bad, "2026-09-10"))
			res.expectCode(http.StatusBadRequest)
			res.expectMessageMentions("tanggal")
		})
	}
}

// A date field left blank is refused by its own validate tag, naming the field
// — the same treatment every other required field gets.
func TestAdminDates_EmptyIsRefusedByName(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	res := h.as("admin").POST("/api/v1/admin/invoices", invoiceFormPayload(""))
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("due_date")

	res = h.as("admin").POST("/api/v1/admin/packages/package-1/batches",
		batchFormPayload("2026-09-01", ""))
	res.expectCode(http.StatusBadRequest)
	res.expectMessageMentions("return_date")
}

// The other two date filters in the interface are query parameters, parsed
// server-side as plain dates already. They are exercised here so "every date in
// the interface" is a claim with a test behind it rather than an assertion.
func TestAdminDateFilters_TakePlainDates(t *testing.T) {
	h := newHarness(t)
	h.seedBaseline()

	from, to := calendar.Today().AddDays(-30).String(), calendar.Today().String()
	h.as("admin").GET("/api/v1/admin/leads?date_from=" + from + "&date_to=" + to).
		expectCode(http.StatusOK)
	h.as("admin").GET("/api/v1/admin/chatbot-logs?date_from=" + from + "&date_to=" + to).
		expectCode(http.StatusOK)
}
