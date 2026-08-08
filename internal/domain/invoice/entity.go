package invoice

import "time"

// Invoice is the billing document for a participant.
//
// The validate tags apply when a request binds into this type (PRD §19.3). The
// status vocabulary mirrors the invoices_status_check constraint as amended by
// db/migrations/006_v2_features.sql; it is omitempty because Service.Create
// issues every invoice as "diterbitkan" regardless of what the caller sent.
type Invoice struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	ParticipantID string    `json:"participant_id" validate:"required"`
	BatchID       string    `json:"batch_id" validate:"required"`
	Amount        float64   `json:"amount" validate:"required,gt=0"`
	DueDate       time.Time `json:"due_date" validate:"required"`
	Status        string    `json:"status" validate:"omitempty,oneof=diterbitkan menunggu_bayar dibayar lunas menunggu_konfirmasi_gateway"`
	PDFPath       string    `json:"pdf_path"`
	Notes         string    `json:"notes"`
	// Midtrans (v2.0 F1) — populated by GetByOrderID; written via SetSnap.
	SnapToken       string     `json:"snap_token,omitempty"`
	MidtransOrderID string     `json:"midtrans_order_id,omitempty"`
	IssuedBy        string     `json:"issued_by"`
	ConfirmedBy     *string    `json:"confirmed_by,omitempty"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	// DeletedAt marks a soft-deleted invoice (§13.1). Every read filters it out;
	// invoice numbering deliberately does not, so a deleted invoice's number is
	// never handed to a later one (§13.7).
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
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
	FilePath      string     `json:"file_path" validate:"required"`
	AmountClaimed float64    `json:"amount_claimed" validate:"required,gt=0"`
	Notes         string     `json:"notes"`
	Status        string     `json:"status" validate:"omitempty,oneof=menunggu disetujui ditolak"`
	ReviewedBy    *string    `json:"reviewed_by,omitempty"`
	ReviewNotes   string     `json:"review_notes"`
	UploadedAt    time.Time  `json:"uploaded_at"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	// DeletedAt marks a soft-deleted proof (§13.1 / ERD §14.1).
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Filter for listing invoices.
type Filter struct {
	Status        *string
	ParticipantID *string
	Page          int
	PerPage       int
}
