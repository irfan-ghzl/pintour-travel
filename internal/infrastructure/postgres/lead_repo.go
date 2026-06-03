package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
)

type leadRepo struct{ db *sql.DB }

func NewLeadRepo(db *sql.DB) lead.Repository { return &leadRepo{db} }

func (r *leadRepo) Create(ctx context.Context, l *lead.Lead) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO leads
		(id,name,phone,email,package_id,batch_id,pax,message,source,status,
		 assigned_to,consent_given,created_at,updated_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
		RETURNING id,created_at,updated_at`,
		l.Name, l.Phone, l.Email, l.PackageID, l.BatchID, l.Pax,
		l.Message, l.Source, l.Status, l.AssignedTo, l.ConsentGiven,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *leadRepo) Update(ctx context.Context, l *lead.Lead) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE leads SET name=$1,phone=$2,email=$3,package_id=$4,batch_id=$5,
		pax=$6,message=$7,source=$8,updated_at=NOW() WHERE id=$9`,
		l.Name, l.Phone, l.Email, l.PackageID, l.BatchID,
		l.Pax, l.Message, l.Source, l.ID)
	return err
}

func (r *leadRepo) GetByID(ctx context.Context, id string) (*lead.Lead, error) {
	var l lead.Lead
	err := r.db.QueryRowContext(ctx, `
		SELECT l.id,l.name,l.phone,COALESCE(l.email,''),l.package_id,l.batch_id,
		l.pax,COALESCE(l.message,''),l.source,l.status,l.assigned_to,
		l.converted_at,l.consent_given,l.created_at,l.updated_at,
		p.name,COALESCE(u.name,'')
		FROM leads l
		JOIN packages p ON p.id=l.package_id
		LEFT JOIN users u ON u.id=l.assigned_to
		WHERE l.id=$1 AND l.deleted_at IS NULL`, id,
	).Scan(&l.ID, &l.Name, &l.Phone, &l.Email, &l.PackageID, &l.BatchID,
		&l.Pax, &l.Message, &l.Source, &l.Status, &l.AssignedTo,
		&l.ConvertedAt, &l.ConsentGiven, &l.CreatedAt, &l.UpdatedAt,
		&l.PackageName, &l.AssigneeName)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *leadRepo) List(ctx context.Context, f lead.Filter) ([]lead.Lead, int, error) {
	where := "WHERE l.deleted_at IS NULL"
	args := []interface{}{}
	i := 1

	if f.Status != nil {
		where += fmt.Sprintf(" AND l.status=$%d", i)
		args = append(args, *f.Status)
		i++
	}
	if f.AssignedTo != nil {
		where += fmt.Sprintf(" AND l.assigned_to=$%d", i)
		args = append(args, *f.AssignedTo)
		i++
	}
	if f.PackageID != nil {
		where += fmt.Sprintf(" AND l.package_id=$%d", i)
		args = append(args, *f.PackageID)
		i++
	}
	if f.Search != nil {
		where += fmt.Sprintf(" AND (l.name ILIKE $%d OR l.phone ILIKE $%d)", i, i+1)
		pat := "%" + *f.Search + "%"
		args = append(args, pat, pat)
		i += 2
	}
	// FR-CRM-05: filter rentang tanggal masuk
	if f.DateFrom != nil {
		where += fmt.Sprintf(" AND l.created_at>=$%d", i)
		args = append(args, *f.DateFrom)
		i++
	}
	if f.DateTo != nil {
		where += fmt.Sprintf(" AND l.created_at<=$%d", i)
		args = append(args, *f.DateTo)
		i++
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM leads l JOIN packages p ON p.id=l.package_id LEFT JOIN users u ON u.id=l.assigned_to "+where,
		args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	offset := (f.Page - 1) * f.PerPage
	args = append(args, f.PerPage, offset)
	q := fmt.Sprintf(`
		SELECT l.id,l.name,l.phone,COALESCE(l.email,''),l.package_id,l.batch_id,
		l.pax,COALESCE(l.message,''),l.source,l.status,l.assigned_to,
		l.converted_at,l.consent_given,l.created_at,l.updated_at,
		p.name,COALESCE(u.name,'')
		FROM leads l
		JOIN packages p ON p.id=l.package_id
		LEFT JOIN users u ON u.id=l.assigned_to
		%s ORDER BY l.created_at DESC LIMIT $%d OFFSET $%d`, where, i, i+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var leads []lead.Lead
	for rows.Next() {
		var l lead.Lead
		if err := rows.Scan(&l.ID, &l.Name, &l.Phone, &l.Email, &l.PackageID,
			&l.BatchID, &l.Pax, &l.Message, &l.Source, &l.Status, &l.AssignedTo,
			&l.ConvertedAt, &l.ConsentGiven, &l.CreatedAt, &l.UpdatedAt,
			&l.PackageName, &l.AssigneeName); err != nil {
			return nil, 0, err
		}
		leads = append(leads, l)
	}
	return leads, total, rows.Err()
}

func (r *leadRepo) UpdateStatus(ctx context.Context, id, status, _ string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET status=$1,updated_at=NOW() WHERE id=$2`, status, id)
	return err
}

func (r *leadRepo) AssignTo(ctx context.Context, leadID, consultantID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET assigned_to=$1,updated_at=NOW() WHERE id=$2`, consultantID, leadID)
	return err
}

func (r *leadRepo) MarkConverted(ctx context.Context, leadID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE leads SET status='peserta',converted_at=$1,updated_at=NOW() WHERE id=$2`, now, leadID)
	return err
}

func (r *leadRepo) CountActiveByConsultant(ctx context.Context, consultantID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM leads WHERE assigned_to=$1 AND status NOT IN ('tidak_deal','peserta')`,
		consultantID).Scan(&n)
	return n, err
}

// ─── Note ─────────────────────────────────────────────────────────────────────

type leadNoteRepo struct{ db *sql.DB }

func NewLeadNoteRepo(db *sql.DB) lead.NoteRepository { return &leadNoteRepo{db} }

func (r *leadNoteRepo) Create(ctx context.Context, n *lead.Note) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO lead_notes(id,lead_id,user_id,note,created_at)
		VALUES(gen_random_uuid(),$1,$2,$3,NOW())
		RETURNING id,created_at`,
		n.LeadID, n.UserID, n.Note,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *leadNoteRepo) ListByLead(ctx context.Context, leadID string) ([]lead.Note, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ln.id,ln.lead_id,ln.user_id,ln.note,ln.created_at,COALESCE(u.name,'')
		FROM lead_notes ln
		LEFT JOIN users u ON u.id=ln.user_id
		WHERE ln.lead_id=$1 ORDER BY ln.created_at`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []lead.Note
	for rows.Next() {
		var n lead.Note
		if err := rows.Scan(&n.ID, &n.LeadID, &n.UserID, &n.Note, &n.CreatedAt, &n.UserName); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
