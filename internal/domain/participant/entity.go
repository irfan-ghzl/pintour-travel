package participant

import "time"

// RoomTypes lists every room a participant may be booked into. Keep in sync
// with the participants_room_type_check constraint in
// db/migrations/003_prd_schema.sql — a value outside this set is refused by the
// column, so it has to be refused before anything is written.
var RoomTypes = []string{"single", "double", "triple"}

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
	BatchID        string    `json:"batch_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
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

// Filter for listing participants.
type Filter struct {
	BatchID  *string
	IsActive *bool
	Search   *string
	Page     int
	PerPage  int
}
