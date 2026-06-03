package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
)

type documentRepo struct{ db *sql.DB }

func NewDocumentRepo(db *sql.DB) document.Repository { return &documentRepo{db} }

func (r *documentRepo) Create(ctx context.Context, d *document.Document) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO documents
		(id,participant_id,document_type,file_path,file_name,status,uploaded_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,'menunggu',NOW())
		RETURNING id,uploaded_at`,
		d.ParticipantID, d.DocumentType, d.FilePath, d.FileName,
	).Scan(&d.ID, &d.UploadedAt)
}

func (r *documentRepo) GetByID(ctx context.Context, id string) (*document.Document, error) {
	var d document.Document
	err := r.db.QueryRowContext(ctx, `
		SELECT d.id,d.participant_id,d.document_type,d.file_path,d.file_name,
		d.status,COALESCE(d.rejection_reason,''),d.reviewed_by,d.uploaded_at,d.reviewed_at,
		COALESCE(p.name,'')
		FROM documents d
		LEFT JOIN participants p ON p.id=d.participant_id
		WHERE d.id=$1`, id,
	).Scan(&d.ID, &d.ParticipantID, &d.DocumentType, &d.FilePath, &d.FileName,
		&d.Status, &d.RejectionReason, &d.ReviewedBy, &d.UploadedAt, &d.ReviewedAt,
		&d.ParticipantName)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *documentRepo) List(ctx context.Context, f document.Filter) ([]document.Document, error) {
	args := []interface{}{}
	q := `SELECT d.id,d.participant_id,d.document_type,d.file_path,d.file_name,
		d.status,COALESCE(d.rejection_reason,''),d.reviewed_by,d.uploaded_at,d.reviewed_at,
		COALESCE(p.name,'')
		FROM documents d
		LEFT JOIN participants p ON p.id=d.participant_id`
	if f.ParticipantID != nil && f.Status != nil {
		q += " WHERE d.participant_id=$1 AND d.status=$2"
		args = append(args, *f.ParticipantID, *f.Status)
	} else if f.ParticipantID != nil {
		q += " WHERE d.participant_id=$1"
		args = append(args, *f.ParticipantID)
	} else if f.Status != nil {
		q += " WHERE d.status=$1"
		args = append(args, *f.Status)
	}
	q += " ORDER BY d.uploaded_at DESC"
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *documentRepo) ListByParticipant(ctx context.Context, participantID string) ([]document.Document, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id,d.participant_id,d.document_type,d.file_path,d.file_name,
		d.status,COALESCE(d.rejection_reason,''),d.reviewed_by,d.uploaded_at,d.reviewed_at,
		COALESCE(p.name,'')
		FROM documents d
		LEFT JOIN participants p ON p.id=d.participant_id
		WHERE d.participant_id=$1 ORDER BY d.document_type,d.uploaded_at`, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *documentRepo) Review(ctx context.Context, id, status, reviewedBy, rejectionReason string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE documents SET status=$1,reviewed_by=$2,rejection_reason=$3,reviewed_at=$4
		WHERE id=$5`, status, reviewedBy, rejectionReason, now, id)
	return err
}

// Delete performs soft delete per §13.1.
func (r *documentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE documents SET deleted_at=NOW() WHERE id=$1`, id)
	return err
}

func scanDocuments(rows *sql.Rows) ([]document.Document, error) {
	var list []document.Document
	for rows.Next() {
		var d document.Document
		if err := rows.Scan(&d.ID, &d.ParticipantID, &d.DocumentType, &d.FilePath,
			&d.FileName, &d.Status, &d.RejectionReason, &d.ReviewedBy,
			&d.UploadedAt, &d.ReviewedAt, &d.ParticipantName); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// ─── CountryRequirement ───────────────────────────────────────────────────────

type countryRequirementRepo struct{ db *sql.DB }

func NewCountryRequirementRepo(db *sql.DB) document.CountryRequirementRepository {
	return &countryRequirementRepo{db}
}

func (r *countryRequirementRepo) List(ctx context.Context, countryCode string) ([]document.CountryRequirement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,country_code,country_name,document_type,is_required,
		COALESCE(description,''),created_at
		FROM country_document_requirements
		WHERE country_code=$1 ORDER BY is_required DESC,document_type`, countryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []document.CountryRequirement
	for rows.Next() {
		var cr document.CountryRequirement
		if err := rows.Scan(&cr.ID, &cr.CountryCode, &cr.CountryName, &cr.DocumentType,
			&cr.IsRequired, &cr.Description, &cr.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, cr)
	}
	return list, rows.Err()
}

func (r *countryRequirementRepo) Create(ctx context.Context, cr *document.CountryRequirement) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO country_document_requirements
		(id,country_code,country_name,document_type,is_required,description,created_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,NOW())
		RETURNING id,created_at`,
		cr.CountryCode, cr.CountryName, cr.DocumentType, cr.IsRequired, cr.Description,
	).Scan(&cr.ID, &cr.CreatedAt)
}

func (r *countryRequirementRepo) Update(ctx context.Context, cr *document.CountryRequirement) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE country_document_requirements
		SET country_name=$1,document_type=$2,is_required=$3,description=$4
		WHERE id=$5`,
		cr.CountryName, cr.DocumentType, cr.IsRequired, cr.Description, cr.ID)
	return err
}

func (r *countryRequirementRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM country_document_requirements WHERE id=$1`, id)
	return err
}
