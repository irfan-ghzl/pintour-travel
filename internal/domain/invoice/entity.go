package invoice

import "time"

// Invoice is the billing document for a participant.
type Invoice struct {
	ID            string     `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	ParticipantID string     `json:"participant_id"`
	BatchID       string     `json:"batch_id"`
	Amount        float64    `json:"amount"`
	DueDate       time.Time  `json:"due_date"`
	Status        string     `json:"status"`
	PDFPath       string     `json:"pdf_path"`
	Notes         string     `json:"notes"`
	IssuedBy      string     `json:"issued_by"`
	ConfirmedBy   *string    `json:"confirmed_by,omitempty"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	// Joined
	ParticipantName  string `json:"participant_name,omitempty"`
	ParticipantPhone string `json:"participant_phone,omitempty"`
	PackageName      string `json:"package_name,omitempty"`
	IssuedByName     string `json:"issued_by_name,omitempty"`
}

// PaymentProof is a transfer receipt uploaded by participant.
type PaymentProof struct {
	ID            string     `json:"id"`
	InvoiceID     string     `json:"invoice_id"`
	FilePath      string     `json:"file_path"`
	AmountClaimed float64    `json:"amount_claimed"`
	Notes         string     `json:"notes"`
	Status        string     `json:"status"`
	ReviewedBy    *string    `json:"reviewed_by,omitempty"`
	ReviewNotes   string     `json:"review_notes"`
	UploadedAt    time.Time  `json:"uploaded_at"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
}

// Filter for listing invoices.
type Filter struct {
	Status        *string
	ParticipantID *string
	Page          int
	PerPage       int
}
