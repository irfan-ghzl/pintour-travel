// Package privacy carries the data-subject rights the PRD commits to in §25.5.
//
// Only erasure lives here. Access and rectification are already served by the
// portal reading and writing the participant's own row; erasure is different
// because it cannot be served by the request that asks for it — it is a promise
// to do something later, to someone who will not be watching, and so it needs a
// record of its own.
package privacy

import (
	"context"
	"errors"
	"time"
)

// Statuses an erasure request moves through. Kept in step with the
// data_deletion_requests_status_check constraint.
const (
	// StatusPending is a request received and not yet acted on. UU PDP Pasal 46
	// gives 14 working days, which is what the portal tells the participant.
	StatusPending = "menunggu"
	// StatusDone means the participant's personal data has been anonymised.
	StatusDone = "selesai"
	// StatusRejected is for a request that must not be honoured — a participant
	// on an upcoming departure whose identity documents the trip still needs.
	StatusRejected = "ditolak"
)

// ErrAlreadyProcessed is returned when a request has already been acted on, so
// two admins working the same queue cannot anonymise twice or reverse a refusal
// by accident.
var ErrAlreadyProcessed = errors.New("permintaan penghapusan sudah diproses")

// DeletionRequest is one participant asking for their personal data to be erased.
type DeletionRequest struct {
	ID            string     `json:"id"`
	ParticipantID string     `json:"participant_id"`
	Reason        string     `json:"reason,omitempty"`
	Status        string     `json:"status"`
	RequestedAt   time.Time  `json:"requested_at"`
	ProcessedBy   *string    `json:"processed_by,omitempty"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	Notes         string     `json:"notes,omitempty"`

	// Joined for the admin queue, so a reviewer sees who is asking without a
	// second lookup. Empty once the participant has been anonymised — which is
	// the point, and worth seeing in the list.
	ParticipantName  string `json:"participant_name,omitempty"`
	ParticipantPhone string `json:"participant_phone,omitempty"`
}

// IsOpen reports whether the request is still waiting on someone.
func (r *DeletionRequest) IsOpen() bool { return r != nil && r.Status == StatusPending }

// DaysWaiting is how long the request has been outstanding, which is the number
// the 14-working-day commitment is measured against.
func (r *DeletionRequest) DaysWaiting() int { return r.DaysWaitingAt(time.Now()) }

// DaysWaitingAt is DaysWaiting relative to a supplied clock, so the count can be
// tested without waiting a day.
func (r *DeletionRequest) DaysWaitingAt(now time.Time) int {
	if r == nil || r.RequestedAt.IsZero() {
		return 0
	}
	end := now
	if r.ProcessedAt != nil {
		end = *r.ProcessedAt
	}
	days := int(end.Sub(r.RequestedAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// Repository stores erasure requests and carries out the erasure itself.
//
// Anonymise sits here rather than on the participant repository because what it
// does is not an edit to a participant: it is the completion of a request, and
// the two have to commit together or a participant is left anonymised with the
// request still open — or, worse, a request closed with the data still there.
type Repository interface {
	// Create records a new request. When the participant already has one open,
	// the existing request is returned unchanged rather than a second one made.
	Create(ctx context.Context, r *DeletionRequest) error
	GetByID(ctx context.Context, id string) (*DeletionRequest, error)
	// List returns requests newest first, optionally narrowed to one status.
	List(ctx context.Context, status string) ([]DeletionRequest, error)
	// Anonymise carries out an approved request: the participant's identifying
	// fields are overwritten, their sensitive documents are soft-deleted, and the
	// request is closed — all in one transaction.
	//
	// §25.4 says participant data is anonymised rather than removed, so the
	// departure statistics it feeds stay whole, and the invoices attached to it
	// stay readable as the financial records they are.
	Anonymise(ctx context.Context, requestID, processedBy, notes string) error
	// Reject closes a request without erasing anything, recording why.
	Reject(ctx context.Context, requestID, processedBy, notes string) error
}
