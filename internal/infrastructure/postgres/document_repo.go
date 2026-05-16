package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
)

// DocumentRepo implements document.Repository using PostgreSQL.
type DocumentRepo struct {
	db *sql.DB
}

// NewDocumentRepo creates a new DocumentRepo.
func NewDocumentRepo(db *sql.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

func (r *DocumentRepo) ListByParticipant(ctx context.Context, participantID string) ([]document.Document, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, participant_id, doc_type, file_url, notes, verified, verified_by, verified_at, uploaded_at
		FROM participant_documents
		WHERE participant_id = $1
		ORDER BY uploaded_at ASC`, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *DocumentRepo) ListByBooking(ctx context.Context, bookingID string) ([]document.Document, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pd.id, pd.participant_id, pd.doc_type, pd.file_url, pd.notes,
		       pd.verified, pd.verified_by, pd.verified_at, pd.uploaded_at
		FROM participant_documents pd
		INNER JOIN booking_participants bp ON bp.id = pd.participant_id
		WHERE bp.booking_id = $1
		ORDER BY bp.full_name, pd.uploaded_at ASC`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func scanDocuments(rows *sql.Rows) ([]document.Document, error) {
	var result []document.Document
	for rows.Next() {
		var d document.Document
		var uploadedAt time.Time
		var verifiedAt *time.Time
		if err := rows.Scan(
			&d.ID, &d.ParticipantID, &d.DocType, &d.FileURL, &d.Notes,
			&d.Verified, &d.VerifiedBy, &verifiedAt, &uploadedAt,
		); err != nil {
			return nil, err
		}
		d.UploadedAt = uploadedAt.Format(time.RFC3339)
		if verifiedAt != nil {
			s := verifiedAt.Format(time.RFC3339)
			d.VerifiedAt = &s
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *DocumentRepo) Create(ctx context.Context, p document.CreateParams) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO participant_documents (participant_id, doc_type, file_url, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		p.ParticipantID, p.DocType, p.FileURL, p.Notes,
	).Scan(&id)
	return id, err
}

func (r *DocumentRepo) Verify(ctx context.Context, id, verifiedByUserID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE participant_documents SET verified=TRUE, verified_by=$2, verified_at=NOW() WHERE id=$1`,
		id, verifiedByUserID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *DocumentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM participant_documents WHERE id=$1`, id)
	return err
}
