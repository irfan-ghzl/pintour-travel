package document

import (
	"encoding/json"
	"time"
)

// Types lists every document kind a participant may upload. Keep in sync with
// the documents_document_type_check constraint in
// db/migrations/003_prd_schema.sql.
var Types = []string{"passport", "ktp", "rekening_koran", "visa_support", "lainnya"}

// Document is a file uploaded by a participant.
//
// The validate tags apply when an upload request binds into this type
// (PRD §19.3); both vocabularies mirror the documents_document_type_check and
// documents_status_check constraints in db/migrations/003_prd_schema.sql.
type Document struct {
	ID              string     `json:"id"`
	ParticipantID   string     `json:"participant_id"`
	DocumentType    string     `json:"document_type" validate:"required,document_type"`
	FilePath        string     `json:"file_path" validate:"required"`
	FileName        string     `json:"file_name" validate:"required"`
	Status          string     `json:"status" validate:"omitempty,oneof=menunggu disetujui ditolak"`
	RejectionReason string     `json:"rejection_reason"`
	ReviewedBy      *string    `json:"reviewed_by,omitempty"`
	UploadedAt      time.Time  `json:"uploaded_at"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	// Joined
	ParticipantName string `json:"participant_name,omitempty"`
}

// CountryRequirement defines documents needed for a destination country.
type CountryRequirement struct {
	ID           string    `json:"id"`
	CountryCode  string    `json:"country_code" validate:"required,max=5"`
	CountryName  string    `json:"country_name" validate:"required"`
	DocumentType string    `json:"document_type" validate:"required,document_type"`
	IsRequired   bool      `json:"is_required"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

// Filter for listing documents.
//
// Page/PerPage were the only list filter in the system without them: the review
// page fetched every document ever uploaded to show twenty, and the dashboard
// fetched all of them to count them.
type Filter struct {
	ParticipantID *string
	Status        *string
	Page          int
	PerPage       int
}

// TODO(ocr-v2.0-F3): OCRResult — aktifkan ketika GCP Vision billing on.
// OCRResult is the stored output of auto-OCR for a document (v2.0 F3).
type OCRResult struct {
	ID               string          `json:"id"`
	DocumentID       string          `json:"document_id"`
	ExtractedData    json.RawMessage `json:"extracted_data"`
	Confidence       float64         `json:"confidence"`
	ValidationPassed bool            `json:"validation_passed"`
	ValidationNotes  string          `json:"validation_notes"`
	CreatedAt        time.Time       `json:"created_at"`
}
