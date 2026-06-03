package airport

import "time"

// Checklist is the airport handling status per participant.
type Checklist struct {
	ID                  string     `json:"id"`
	ParticipantID       string     `json:"participant_id"`
	BatchID             string     `json:"batch_id"`
	BaggageChecked      bool       `json:"baggage_checked"`
	BaggageCheckedAt    *time.Time `json:"baggage_checked_at,omitempty"`
	TicketDistributed   bool       `json:"ticket_distributed"`
	TicketDistributedAt *time.Time `json:"ticket_distributed_at,omitempty"`
	PassportReturned    bool       `json:"passport_returned"`
	PassportReturnedAt  *time.Time `json:"passport_returned_at,omitempty"`
	HandledBy           *string    `json:"handled_by,omitempty"`
	Notes               string     `json:"notes"`
	UpdatedAt           time.Time  `json:"updated_at"`
	// Joined
	ParticipantName  string  `json:"participant_name,omitempty"`
	ParticipantPhone string  `json:"participant_phone,omitempty"`
	HandledByName    *string `json:"handled_by_name,omitempty"`
}

// BatchProgress is a summary of handling progress for one batch.
type BatchProgress struct {
	BatchID      string `json:"batch_id"`
	TotalPax     int    `json:"total_pax"`
	DoneCount    int    `json:"done_count"`
	PendingCount int    `json:"pending_count"`
}

// Filter for listing airport checklists.
type Filter struct {
	BatchID string
	Status  *string // "pending" | "done" | "" (all)
}
