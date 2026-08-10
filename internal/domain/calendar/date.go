// Package calendar holds Date: a day on the calendar, with no time of day and
// no timezone.
//
// It exists because three fields in this system are dates and were modelled as
// instants. A due date, a departure date and a return date are days people
// agree on, not moments: "15 August" is the same day in Jakarta and in Jeddah,
// and attaching midnight-in-some-zone to it invents a precision the fact does
// not have — precision that then shifts the day by one whenever it is read back
// in a different zone.
//
// The database already agreed: due_date, departure_date and return_date are all
// DATE columns. This type is what makes the Go side say the same thing.
package calendar

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Layout is the wire and storage form of a Date — the one `<input type="date">`
// produces and the one Postgres DATE reads.
const Layout = "2006-01-02"

// Date is a calendar day.
type Date struct {
	// t is midnight UTC on the day. UTC is not a claim about where the day
	// happened; it is the fixed reference that keeps the day from drifting when
	// the value is converted, compared or stored.
	t time.Time
}

// New builds a Date from its parts.
func New(year int, month time.Month, day int) Date {
	return Date{t: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// Of is the calendar day t falls on, in t's own location — the day someone
// standing where t was recorded would call it.
func Of(t time.Time) Date {
	return New(t.Year(), t.Month(), t.Day())
}

// Today is the current calendar day in the local timezone.
func Today() Date { return Of(time.Now()) }

// Parse reads a date in Layout form.
func Parse(s string) (Date, error) {
	t, err := time.Parse(Layout, s)
	if err != nil {
		return Date{}, err
	}
	return Date{t: t}, nil
}

// IsZero reports whether the date was never set.
func (d Date) IsZero() bool { return d.t.IsZero() }

// Time is midnight UTC on the day, for the callers that still need a time.Time
// — rendering, and arithmetic against an instant.
func (d Date) Time() time.Time { return d.t }

// String renders the date in Layout form.
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.t.Format(Layout)
}

// Format renders the date with a time package layout, for the PDFs and messages
// that spell months out. An unset date renders as nothing rather than as the
// year zero — "01 January 0001" on an invoice is worse than a blank.
func (d Date) Format(layout string) string {
	if d.IsZero() {
		return ""
	}
	return d.t.Format(layout)
}

// AddDays returns the date n days later; n may be negative.
func (d Date) AddDays(n int) Date { return Date{t: d.t.AddDate(0, 0, n)} }

// Before, After and Equal compare two calendar days.
func (d Date) Before(other Date) bool { return d.t.Before(other.t) }
func (d Date) After(other Date) bool  { return d.t.After(other.t) }
func (d Date) Equal(other Date) bool  { return d.t.Equal(other.t) }

// DaysUntil counts the days from d to other; negative once other has passed.
func (d Date) DaysUntil(other Date) int {
	return int(other.t.Sub(d.t).Hours() / 24)
}

// ParseError is what a request carrying an unreadable date fails with. It is a
// named type so the HTTP layer can tell it apart from any other decode failure
// and answer with a message about the date, rather than the generic "format
// tidak valid" that told an admin nothing about which field to fix.
type ParseError struct {
	Value string
}

func (e ParseError) Error() string {
	if e.Value == "" {
		return fmt.Sprintf("tanggal harus berupa teks dengan format %s", Layout)
	}
	return fmt.Sprintf("tanggal %q tidak dikenali, gunakan format %s (contoh: 2026-08-15)",
		e.Value, Layout)
}

// MarshalJSON writes the date as "YYYY-MM-DD", or null when unset.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON accepts what the admin forms actually send — a plain
// "YYYY-MM-DD" — and, for older clients and stored payloads, a full RFC3339
// timestamp, of which it keeps the day.
//
// The error names the value it could not read. A decode failure used to reach
// the admin as a generic "format tidak valid" with no hint that the date was
// the problem, which is how a form that could never succeed went unnoticed.
func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		// A JSON null or a missing field leaves the date unset rather than
		// failing: whether a date is required is the validate tag's business,
		// not the decoder's.
		if string(b) == "null" {
			*d = Date{}
			return nil
		}
		return ParseError{}
	}
	if s = strings.TrimSpace(s); s == "" {
		*d = Date{}
		return nil
	}
	if parsed, err := Parse(s); err == nil {
		*d = parsed
		return nil
	}
	// RFC3339 is accepted too, for clients written against the older contract
	// and for payloads already stored somewhere. The day is taken as UTC, which
	// is the day the value names when it carries no other information.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		*d = Of(t.UTC())
		return nil
	}
	return ParseError{Value: s}
}

// Value stores the date as a DATE column.
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.t, nil
}

// Scan reads a DATE column back. Postgres hands it over as a time.Time, in
// whatever location the driver chose; the day is taken from that reading rather
// than converted, so a driver returning local time does not move the date.
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*d = Date{}
		return nil
	case time.Time:
		*d = Of(v)
		return nil
	case string:
		parsed, err := Parse(v)
		if err != nil {
			return fmt.Errorf("scan tanggal %q: %w", v, err)
		}
		*d = parsed
		return nil
	case []byte:
		return d.Scan(string(v))
	}
	return fmt.Errorf("tidak bisa membaca %T sebagai tanggal", src)
}
