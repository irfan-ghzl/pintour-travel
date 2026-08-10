package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
)

type ocrResultRepo struct{ db *sql.DB }

func NewOCRResultRepo(db *sql.DB) document.OCRResultRepository { return &ocrResultRepo{db} }

func (r *ocrResultRepo) Create(ctx context.Context, o *document.OCRResult) error {
	data := o.ExtractedData
	if len(data) == 0 {
		data = []byte("{}")
	}
	return r.db.QueryRowContext(ctx, `
		INSERT INTO document_ocr_results
		(id, document_id, extracted_data, confidence, validation_passed, validation_notes, created_at)
		VALUES (gen_random_uuid(), $1, $2::jsonb, $3, $4, $5, NOW())
		RETURNING id, created_at`,
		o.DocumentID, string(data), o.Confidence, o.ValidationPassed, o.ValidationNotes,
	).Scan(&o.ID, &o.CreatedAt)
}

func (r *ocrResultRepo) GetByDocument(ctx context.Context, documentID string) (*document.OCRResult, error) {
	var o document.OCRResult
	var data []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, document_id, extracted_data, confidence, validation_passed,
		COALESCE(validation_notes,''), created_at
		FROM document_ocr_results WHERE document_id=$1
		ORDER BY created_at DESC LIMIT 1`, documentID,
	).Scan(&o.ID, &o.DocumentID, &data, &o.Confidence, &o.ValidationPassed, &o.ValidationNotes, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	o.ExtractedData = data
	return &o, nil
}

func (r *ocrResultRepo) LatestPassportExpiry(ctx context.Context, participantID string) (string, error) {
	var expiry sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT o.extracted_data->>'expiry_date'
		FROM document_ocr_results o
		JOIN documents d ON d.id = o.document_id
		WHERE d.participant_id = $1
		  AND d.document_type IN ('passport','paspor')
		  AND COALESCE(o.extracted_data->>'expiry_date','') <> ''
		ORDER BY o.created_at DESC
		LIMIT 1`, participantID,
	).Scan(&expiry)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return expiry.String, nil
}
