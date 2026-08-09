package invoice

import (
	"errors"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// ErrAlreadySettled is returned when an invoice that is already paid in full is
// confirmed again. It is not a failure the admin caused — the money arrived,
// twice-confirming it just must not send a second round of notifications.
var ErrAlreadySettled = errors.New("invoice sudah lunas")

// StatusSettled is the terminal invoice status; UnsettledStatuses are the ones
// that still owe money and can therefore fall overdue. Keep in sync with the
// invoices_status_check constraint as amended by
// db/migrations/006_v2_features.sql.
const StatusSettled = "lunas"

var UnsettledStatuses = []string{"diterbitkan", "menunggu_bayar", "dibayar"}

// Invoice is the billing document for a participant.
//
// The validate tags apply when a request binds into this type (PRD §19.3). The
// status vocabulary mirrors the invoices_status_check constraint as amended by
// db/migrations/006_v2_features.sql; it is omitempty because Service.Create
// issues every invoice as "diterbitkan" regardless of what the caller sent.
type Invoice struct {
	ID            string        `json:"id"`
	InvoiceNumber string        `json:"invoice_number"`
	ParticipantID string        `json:"participant_id" validate:"required"`
	BatchID       string        `json:"batch_id" validate:"required"`
	Amount        float64       `json:"amount" validate:"required,gt=0"`
	DueDate       calendar.Date `json:"due_date" validate:"required"`
	Status        string        `json:"status" validate:"omitempty,oneof=diterbitkan menunggu_bayar dibayar lunas menunggu_konfirmasi_gateway"`
	PDFPath       string        `json:"pdf_path"`
	Notes         string        `json:"notes"`
	// Midtrans (v2.0 F1) — the session currently open for this invoice, written by
	// SetSnap. Which invoice a notification belongs to is resolved through the
	// gateway_orders table, not through these columns: a payment completed in a
	// tab opened earlier names an order this invoice no longer points at.
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

// ConfirmPayment marks the invoice paid in full by confirmedBy at time at
// (§14.4).
//
// An invoice already settled is refused rather than confirmed twice: the second
// confirmation would overwrite who settled it and when, and every path that
// confirms also notifies the participant.
//
// Writing the row and activating the portal stay in the application layer —
// they are two tables and a WhatsApp message, none of which an entity can do.
func (i *Invoice) ConfirmPayment(confirmedBy string, at time.Time) error {
	if i == nil {
		return errors.New("invoice tidak boleh kosong")
	}
	if confirmedBy == "" {
		return errors.New("pelaku konfirmasi harus diisi")
	}
	if i.Status == StatusSettled {
		return ErrAlreadySettled
	}
	i.Status = StatusSettled
	i.ConfirmedBy = &confirmedBy
	i.ConfirmedAt = &at
	return nil
}

// IsOverdue reports whether the invoice is past its due date with money still
// owed (§14.4).
func (i *Invoice) IsOverdue() bool { return i.IsOverdueAt(time.Now()) }

// IsOverdueAt is IsOverdue judged at a given moment.
//
// A settled invoice is never overdue, and neither is one waiting on the gateway
// to confirm a payment the participant has already made — chasing either would
// be dunning someone who has paid. A soft-deleted invoice is nobody's debt.
func (i *Invoice) IsOverdueAt(now time.Time) bool {
	if i == nil || i.DeletedAt != nil || i.DueDate.IsZero() {
		return false
	}
	// The due date is a whole day, so an invoice due today is not overdue until
	// today is over — the participant has the rest of the day to pay.
	if !i.DueDate.Before(calendar.Of(now)) {
		return false
	}
	for _, s := range UnsettledStatuses {
		if i.Status == s {
			return true
		}
	}
	return false
}

// RemainingBalance is what is still owed once paid has been credited, never
// below zero — an overpaid invoice owes nothing, not a negative amount.
func (i *Invoice) RemainingBalance(paid float64) float64 {
	if i == nil {
		return 0
	}
	if remaining := i.Amount - paid; remaining > 0 {
		return remaining
	}
	return 0
}

// IsFullyPaid reports whether paid covers the invoice. It is the one place the
// "covered" rule is stated: the settle path and the gateway path used to decide
// it separately, and a payment that counted as full in one and partial in the
// other is how an invoice ends up settled for less than it asked for.
func (i *Invoice) IsFullyPaid(paid float64) bool {
	return i != nil && paid >= i.Amount
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
