package lead

import "time"

// Lead is the CRM aggregate representing a prospective customer.
type Lead struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	PackageID    string     `json:"package_id"`
	BatchID      *string    `json:"batch_id,omitempty"`
	Pax          int        `json:"pax"`
	Message      string     `json:"message"`
	Source       string     `json:"source"`
	Status       string     `json:"status"`
	AssignedTo   *string    `json:"assigned_to,omitempty"`
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
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UserName  string    `json:"user_name,omitempty"`
}

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
