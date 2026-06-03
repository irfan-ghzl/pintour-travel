package document

import "time"

// Document is a file uploaded by a participant.
type Document struct {
	ID              string     `json:"id"`
	ParticipantID   string     `json:"participant_id"`
	DocumentType    string     `json:"document_type"`
	FilePath        string     `json:"file_path"`
	FileName        string     `json:"file_name"`
	Status          string     `json:"status"`
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
	CountryCode  string    `json:"country_code"`
	CountryName  string    `json:"country_name"`
	DocumentType string    `json:"document_type"`
	IsRequired   bool      `json:"is_required"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

// Filter for listing documents.
type Filter struct {
	ParticipantID *string
	Status        *string
}
