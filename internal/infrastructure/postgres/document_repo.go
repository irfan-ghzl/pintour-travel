package postgres

import (
	"context"
	"database/sql"
	"fmt"
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
		WHERE d.id=$1 AND d.deleted_at IS NULL`, id,
	).Scan(&d.ID, &d.ParticipantID, &d.DocumentType, &d.FilePath, &d.FileName,
		&d.Status, &d.RejectionReason, &d.ReviewedBy, &d.UploadedAt, &d.ReviewedAt,
		&d.ParticipantName)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *documentRepo) List(ctx context.Context, f document.Filter) ([]document.Document, int, error) {
	// One where-clause, built once and used by both the count and the page, so
	// the total can never describe a different set from the rows.
	where := "WHERE d.deleted_at IS NULL"
	args := []interface{}{}
	if f.ParticipantID != nil {
		args = append(args, *f.ParticipantID)
		where += fmt.Sprintf(" AND d.participant_id=$%d", len(args))
	}
	if f.Status != nil {
		args = append(args, *f.Status)
		where += fmt.Sprintf(" AND d.status=$%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM documents d "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	q := fmt.Sprintf(`SELECT d.id,d.participant_id,d.document_type,d.file_path,d.file_name,
		d.status,COALESCE(d.rejection_reason,''),d.reviewed_by,d.uploaded_at,d.reviewed_at,
		COALESCE(p.name,'')
		FROM documents d
		LEFT JOIN participants p ON p.id=d.participant_id
		%s ORDER BY d.uploaded_at DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list, err := scanDocuments(rows)
	return list, total, err
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

// SummaryByParticipants counts each participant's documents by status in one
// aggregate.
//
// The review queue shows "N of M approved" beside every row. That figure
// describes the participant, so it cannot be counted from the rows on screen —
// those are one page of one status. Nor can it be a query per row: the queue is
// paginated and would issue as many queries as the page is long, for a number
// that does not depend on the page at all.
func (r *documentRepo) SummaryByParticipants(ctx context.Context, participantIDs []string) (map[string]document.StatusSummary, error) {
	out := map[string]document.StatusSummary{}
	if len(participantIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT participant_id, status, COUNT(*)
		FROM documents
		WHERE deleted_at IS NULL AND participant_id = ANY($1::uuid[])
		GROUP BY participant_id, status`, participantIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, status string
		var n int
		if err := rows.Scan(&id, &status, &n); err != nil {
			return nil, err
		}
		summary := out[id]
		summary.AddN(status, n)
		out[id] = summary
	}
	return out, rows.Err()
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
	list := []document.Document{}
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
	// An empty countryCode means "list every requirement" (admin global view);
	// a non-empty one filters to that country (public per-country view).
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,country_code,country_name,document_type,is_required,
		COALESCE(description,''),created_at
		FROM country_document_requirements
		WHERE ($1 = '' OR country_code = $1)
		ORDER BY country_code,is_required DESC,document_type`, countryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []document.CountryRequirement{}
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
