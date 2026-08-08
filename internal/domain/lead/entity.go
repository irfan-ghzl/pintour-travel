package lead

import (
	"errors"
	"time"
)

// ErrInvalidStatus is returned when a status transition target is not one of the
// allowed lead statuses (mirrors the leads_status_check DB constraint).
var ErrInvalidStatus = errors.New("status lead tidak valid")

// Statuses lists every valid lead status, in pipeline order. Keep in sync with
// the leads_status_check constraint in db/migrations/003_prd_schema.sql.
var Statuses = []string{"baru", "dihubungi", "konsultasi", "deal", "tidak_deal", "peserta"}

// IsValidStatus reports whether s is an allowed lead status.
func IsValidStatus(s string) bool {
	for _, v := range Statuses {
		if v == s {
			return true
		}
	}
	return false
}

// Lead is the CRM aggregate representing a prospective customer.
//
// The validate tags apply when the public lead form binds a request into this
// type (PRD §19.3); their vocabularies mirror the leads_source_check and
// leads_status_check constraints in db/migrations/003_prd_schema.sql.
type Lead struct {
	ID           string     `json:"id"`
	Name         string     `json:"name" validate:"required"`
	Phone        string     `json:"phone" validate:"required,phone_id"`
	Email        string     `json:"email" validate:"omitempty,email"`
	PackageID    string     `json:"package_id" validate:"required"`
	BatchID      *string    `json:"batch_id,omitempty"`
	Pax          int        `json:"pax" validate:"omitempty,gte=1,lte=50"`
	Message      string     `json:"message"`
	Source       string     `json:"source" validate:"omitempty,oneof=meta_ads organic referral walk_in"`
	Status       string     `json:"status" validate:"omitempty,lead_status"`
	AssignedTo   *string    `json:"assigned_to,omitempty"`
	PortalUserID *string    `json:"portal_user_id,omitempty"` // v2.0 F4 — set when lead comes from a returning customer
	IsReturning  bool       `json:"is_returning"`             // v2.0 F4 — phone already has a portal account
	ConvertedAt  *time.Time `json:"converted_at,omitempty"`
	ConsentGiven bool       `json:"consent_given"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	// Joined
	PackageName  string  `json:"package_name,omitempty"`
	AssigneeName *string `json:"assignee_name,omitempty"`
}

// Note is a consultant note on a lead.
type Note struct {
	ID        string    `json:"id"`
	LeadID    string    `json:"lead_id"`
	UserID    string    `json:"user_id"`
	Note      string    `json:"note" validate:"required"`
	CreatedAt time.Time `json:"created_at"`
	UserName  string    `json:"user_name,omitempty"`
}

// StatusChange is one recorded transition of a lead's status (FR-CRM-02).
//
// It sits beside Note rather than inside it: a consultant's note is something a
// person chose to write, a status change is something that happened. Folding the
// second into the first — which is what the synthetic "[SISTEM] Status diubah"
// note used to do — loses the distinction and cannot record the previous status.
type StatusChange struct {
	ID         string `json:"id"`
	LeadID     string `json:"lead_id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	// ChangedBy is the staff user who made the change, empty when the scheduler
	// did (see SystemActor).
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
	// Joined
	ChangedByName string `json:"changed_by_name,omitempty"`
}

// SystemActor is the name reported for a change nobody made by hand — the
// nightly job that expires stale leads. Stored as a NULL actor; rendered as this
// so the trail reads the same whoever is looking at it.
const SystemActor = "Sistem"

// Filter for listing leads.
type Filter struct {
	Status     *string
	AssignedTo *string
	PackageID  *string
	Search     *string
	DateFrom   *time.Time
	DateTo     *time.Time
	Page       int
	PerPage    int
}
