package participant

import (
	"errors"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// RoomTypes lists every room a participant may be booked into. Keep in sync
// with the participants_room_type_check constraint in
// db/migrations/003_prd_schema.sql — a value outside this set is refused by the
// column, so it has to be refused before anything is written.
var RoomTypes = []string{"single", "double", "triple"}

// ErrInvalidRoomType is returned when a room outside RoomTypes is requested.
var ErrInvalidRoomType = errors.New("tipe kamar tidak dikenal (pilih: " +
	strings.Join(RoomTypes, ", ") + ")")

// IsValidRoomType reports whether roomType is one the schema accepts.
func IsValidRoomType(roomType string) bool {
	for _, t := range RoomTypes {
		if t == roomType {
			return true
		}
	}
	return false
}

// Participant is an active customer in a departure batch.
type Participant struct {
	ID             string    `json:"id"`
	LeadID         *string   `json:"lead_id,omitempty"`
	PortalUserID   *string   `json:"portal_user_id,omitempty"` // v2.0 F1 — central portal identity
	BatchID        string    `json:"batch_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	NIK            string    `json:"nik,omitempty"` // TODO(ocr-v2.0-F3): auto-filled by OCR when active
	RoomType       string    `json:"room_type" validate:"omitempty,room_type"`
	PortalPassword string    `json:"-"` // jangan serialize password
	IsActive       bool      `json:"is_active"`
	BriefingViewed bool      `json:"briefing_viewed"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// Joined
	BatchDepartureDate *time.Time `json:"batch_departure_date,omitempty"`
	PackageName        string     `json:"package_name,omitempty"`
}

// ActivatePortal opens the participant's portal and returns the identifier they
// sign in with (§14.4).
//
// The password is not issued here. A returning customer keeps the login their
// existing portal identity already carries, so activation is about access, not
// credentials — which is why what comes back is the phone number the activation
// message tells them to use, not a new secret.
//
// Persisting the flag is the repository's job; this is the same change made to
// the entity in hand.
func (p *Participant) ActivatePortal() string {
	if p == nil {
		return ""
	}
	p.IsActive = true
	return p.Phone
}

// HasDeparture reports whether the participant's batch has a departure date at
// all — the case DaysUntilDeparture cannot distinguish from "leaving today".
func (p *Participant) HasDeparture() bool {
	return p != nil && p.BatchDepartureDate != nil
}

// DaysUntilDeparture is the whole days left before the participant flies
// (§14.4). Zero when there is no batch date, which callers separate from a real
// zero with HasDeparture.
func (p *Participant) DaysUntilDeparture() int {
	return p.DaysUntilDepartureFrom(time.Now())
}

// DaysUntilDepartureFrom is DaysUntilDeparture measured from a given moment, so
// the countdown can be tested without waiting for a calendar.
//
// Whole calendar days, the same arithmetic PackageBatch uses: a departure is a
// date, so "how many days left" is a difference between days. Measuring the
// instants instead made the number a participant sees depend on what time of
// day they opened the portal — the same trip reading 14 in the morning and 13
// after lunch.
func (p *Participant) DaysUntilDepartureFrom(now time.Time) int {
	if !p.HasDeparture() {
		return 0
	}
	return calendar.Of(now).DaysUntil(calendar.Of(*p.BatchDepartureDate))
}

// Filter for listing participants.
type Filter struct {
	BatchID  *string
	IsActive *bool
	Search   *string
	// AssignedTo scopes the list to participants whose originating lead is
	// assigned to this consultant (used to limit konsultan to their own).
	AssignedTo *string
	Page       int
	PerPage    int
}
