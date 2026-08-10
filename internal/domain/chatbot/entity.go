package chatbot

import "time"

// Log is a single chatbot conversation turn (v2.0 F2).
type Log struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"` // "user" | "assistant"
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Conversation is the per-phone summary shown in the admin list.
type Conversation struct {
	Phone        string    `json:"phone"`
	MessageCount int       `json:"message_count"`
	FirstChat    time.Time `json:"first_chat"`
	LastChat     time.Time `json:"last_chat"`
	LeadID       *string   `json:"lead_id,omitempty"`
}

// Filter for listing conversations.
type Filter struct {
	Phone    string
	DateFrom string // YYYY-MM-DD, optional
	DateTo   string // YYYY-MM-DD, optional
	Page     int
	Limit    int
}
