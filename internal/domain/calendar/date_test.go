package calendar

import (
	"encoding/json"
	"testing"
	"time"
)

// The payload the admin invoice and batch forms actually send is accepted.
func TestUnmarshalAcceptsFormPayload(t *testing.T) {
	cases := map[string]string{
		`"2026-08-15"`:                "2026-08-15",
		`"2026-01-01"`:                "2026-01-01",
		`"2026-12-31"`:                "2026-12-31",
		`" 2026-08-15 "`:              "2026-08-15", // whitespace tolerated
		`"2026-08-15T00:00:00Z"`:      "2026-08-15", // older RFC3339 clients
		`"2026-08-15T17:30:00Z"`:      "2026-08-15",
		`"2026-08-15T00:00:00+07:00"`: "2026-08-14", // that instant is the 14th in UTC
		`null`:                        "",
		`""`:                          "",
	}
	for payload, want := range cases {
		var d Date
		if err := json.Unmarshal([]byte(payload), &d); err != nil {
			t.Errorf("Unmarshal(%s): %v", payload, err)
			continue
		}
		if got := d.String(); got != want {
			t.Errorf("Unmarshal(%s) = %q, want %q", payload, got, want)
		}
	}
}

// A value that is not a date is refused with a message that says so, naming the
// value — not a generic decode failure.
func TestUnmarshalRejectsNonDates(t *testing.T) {
	for _, payload := range []string{`"15 Agustus 2026"`, `"2026/08/15"`, `"besok"`, `12345`, `{}`} {
		var d Date
		err := json.Unmarshal([]byte(payload), &d)
		if err == nil {
			t.Errorf("Unmarshal(%s) diterima, want error", payload)
			continue
		}
		if !containsAny(err.Error(), "tanggal") {
			t.Errorf("Unmarshal(%s) error = %q, tidak menyebut tanggal", payload, err)
		}
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) <= len(s) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// A date written and read back is the same day, whatever timezone is in play.
// This is the shift the ticket exists to rule out.
func TestRoundTripDoesNotShiftDay(t *testing.T) {
	zones := []*time.Location{time.UTC}
	for _, name := range []string{"Asia/Jakarta", "Asia/Jayapura", "America/Los_Angeles", "Pacific/Kiritimati"} {
		if loc, err := time.LoadLocation(name); err == nil {
			zones = append(zones, loc)
		}
	}

	for _, in := range []string{"2026-01-01", "2026-08-15", "2026-12-31"} {
		var d Date
		if err := json.Unmarshal([]byte(`"`+in+`"`), &d); err != nil {
			t.Fatalf("Unmarshal(%q): %v", in, err)
		}

		// Through JSON.
		out, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if string(out) != `"`+in+`"` {
			t.Errorf("JSON round trip: %s, want %q", out, in)
		}

		// Through the database. A DATE column carries no instant, so a driver
		// hands it back as midnight on that day in whichever location it uses —
		// pgx picks UTC, lib/pq picks the connection's local zone. Both name the
		// same day, and both must read back as that day.
		stored, err := d.Value()
		if err != nil {
			t.Fatalf("Value: %v", err)
		}
		day := stored.(time.Time)
		for _, loc := range zones {
			midnightThere := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
			var back Date
			if err := back.Scan(midnightThere); err != nil {
				t.Fatalf("Scan in %v: %v", loc, err)
			}
			if back.String() != in {
				t.Errorf("DB round trip via %v = %q, want %q", loc, back.String(), in)
			}
		}

		// And as the text form some drivers return instead.
		var fromText Date
		if err := fromText.Scan(in); err != nil || fromText.String() != in {
			t.Errorf("Scan(%q) = %q (%v)", in, fromText.String(), err)
		}
	}
}

func TestZeroDate(t *testing.T) {
	var d Date
	if !d.IsZero() {
		t.Error("Date kosong tidak dilaporkan zero")
	}
	if d.String() != "" {
		t.Errorf("String() = %q, want kosong", d.String())
	}
	out, err := json.Marshal(d)
	if err != nil || string(out) != "null" {
		t.Errorf("Marshal = %s (%v), want null", out, err)
	}
	v, err := d.Value()
	if err != nil || v != nil {
		t.Errorf("Value = %v (%v), want nil", v, err)
	}
}

func TestDateArithmetic(t *testing.T) {
	d := New(2026, time.August, 15)

	if got := d.AddDays(7).String(); got != "2026-08-22" {
		t.Errorf("AddDays(7) = %q, want 2026-08-22", got)
	}
	if got := d.AddDays(-15).String(); got != "2026-07-31" {
		t.Errorf("AddDays(-15) = %q, want 2026-07-31", got)
	}
	if got := d.AddDays(0); !got.Equal(d) {
		t.Errorf("AddDays(0) = %q, want %q", got, d)
	}
	if got := d.DaysUntil(New(2026, time.August, 29)); got != 14 {
		t.Errorf("DaysUntil = %d, want 14", got)
	}
	if got := d.DaysUntil(New(2026, time.August, 14)); got != -1 {
		t.Errorf("DaysUntil masa lalu = %d, want -1", got)
	}
	if !d.Before(New(2026, time.August, 16)) {
		t.Error("15 Agustus tidak dilaporkan sebelum 16 Agustus")
	}
	if !d.After(New(2026, time.August, 14)) {
		t.Error("15 Agustus tidak dilaporkan sesudah 14 Agustus")
	}
}

// Of takes the day as read where the instant was recorded, so a departure
// stamped late at night in Jakarta is still that day and not the next.
func TestOfKeepsLocalDay(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skip("tz database tidak tersedia")
	}
	late := time.Date(2026, 8, 15, 23, 30, 0, 0, jakarta)
	if got := Of(late).String(); got != "2026-08-15" {
		t.Errorf("Of(23:30 WIB) = %q, want 2026-08-15", got)
	}
	early := time.Date(2026, 8, 15, 0, 30, 0, 0, jakarta)
	if got := Of(early).String(); got != "2026-08-15" {
		t.Errorf("Of(00:30 WIB) = %q, want 2026-08-15", got)
	}
}

func TestScan(t *testing.T) {
	var d Date
	if err := d.Scan("2026-08-15"); err != nil || d.String() != "2026-08-15" {
		t.Errorf("Scan(string) = %q (%v)", d.String(), err)
	}
	if err := d.Scan([]byte("2026-08-16")); err != nil || d.String() != "2026-08-16" {
		t.Errorf("Scan([]byte) = %q (%v)", d.String(), err)
	}
	if err := d.Scan(nil); err != nil || !d.IsZero() {
		t.Errorf("Scan(nil) = %q (%v), want zero", d.String(), err)
	}
	if err := d.Scan(42); err == nil {
		t.Error("Scan(int) diterima, want error")
	}
}
