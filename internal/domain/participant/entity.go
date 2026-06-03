package participant

import "time"

// Participant is an active customer in a departure batch.
type Participant struct {
	ID             string    `json:"id"`
	LeadID         *string   `json:"lead_id,omitempty"`
	BatchID        string    `json:"batch_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	RoomType       string    `json:"room_type"`
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
