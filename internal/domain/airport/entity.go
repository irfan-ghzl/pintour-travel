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

// IsComplete reports whether all three handling steps are done for this
// participant (§14.4). It is the definition of "done" the batch progress
// summary counts and the checklist filter selects on.
func (c *Checklist) IsComplete() bool {
	return c != nil && c.BaggageChecked && c.TicketDistributed && c.PassportReturned
}

// MarkBaggage records that the participant's baggage was checked in by staff
// member `by` at time `at` (§14.4).
//
// The first timestamp stands: it is when the step actually happened, and a
// second tap on the same row at the handling desk must not rewrite history. Who
// handled it is updated, because it names whoever touched the row last, which
// is what the column has always meant.
func (c *Checklist) MarkBaggage(by string, at time.Time) {
	c.BaggageChecked, c.BaggageCheckedAt = true, firstStamp(c.BaggageCheckedAt, at)
	c.handledBy(by, at)
}

// MarkTicket records that the participant's ticket was handed over (§14.4).
func (c *Checklist) MarkTicket(by string, at time.Time) {
	c.TicketDistributed, c.TicketDistributedAt = true, firstStamp(c.TicketDistributedAt, at)
	c.handledBy(by, at)
}

// MarkPassport records that the participant's passport was returned (§14.4).
func (c *Checklist) MarkPassport(by string, at time.Time) {
	c.PassportReturned, c.PassportReturnedAt = true, firstStamp(c.PassportReturnedAt, at)
	c.handledBy(by, at)
}

func (c *Checklist) handledBy(by string, at time.Time) {
	c.HandledBy = &by
	c.UpdatedAt = at
}

// firstStamp keeps the moment a step was first completed.
func firstStamp(existing *time.Time, at time.Time) *time.Time {
	if existing != nil {
		return existing
	}
	return &at
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
