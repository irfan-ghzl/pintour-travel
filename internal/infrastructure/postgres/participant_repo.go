package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

type participantRepo struct{ db dbtx }

func NewParticipantRepo(db *sql.DB) participant.Repository { return &participantRepo{db} }

const participantCols = `
	p.id, p.lead_id, p.batch_id, p.name, p.phone, COALESCE(p.email,''),
	p.room_type, p.portal_password, p.is_active, p.briefing_viewed,
	p.created_at, p.updated_at,
	pb.departure_date, pkg.name`

func scanParticipant(s interface{ Scan(...interface{}) error }, pt *participant.Participant) error {
	return s.Scan(
		&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone, &pt.Email,
		&pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
		&pt.CreatedAt, &pt.UpdatedAt,
		&pt.BatchDepartureDate, &pt.PackageName,
	)
}

func (r *participantRepo) Create(ctx context.Context, p *participant.Participant) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO participants
		(id,lead_id,batch_id,name,phone,email,room_type,portal_password,is_active,created_at,updated_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())
		RETURNING id,created_at,updated_at`,
		p.LeadID, p.BatchID, p.Name, p.Phone, p.Email, p.RoomType, p.PortalPassword, p.IsActive,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *participantRepo) Update(ctx context.Context, p *participant.Participant) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE participants SET name=$1,phone=$2,email=$3,room_type=$4,updated_at=NOW()
		WHERE id=$5`,
		p.Name, p.Phone, p.Email, p.RoomType, p.ID)
	return err
}

func (r *participantRepo) GetByID(ctx context.Context, id string) (*participant.Participant, error) {
	var pt participant.Participant
	err := r.db.QueryRowContext(ctx, `
		SELECT `+participantCols+`
		FROM participants p
		JOIN package_batches pb ON pb.id=p.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		WHERE p.id=$1`, id,
	).Scan(&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone, &pt.Email,
		&pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
		&pt.CreatedAt, &pt.UpdatedAt, &pt.BatchDepartureDate, &pt.PackageName)
	if err != nil {
		return nil, err
	}
	return &pt, nil
}

func (r *participantRepo) GetByPhone(ctx context.Context, phone string) (*participant.Participant, error) {
	var pt participant.Participant
	err := r.db.QueryRowContext(ctx, `
		SELECT `+participantCols+`
		FROM participants p
		JOIN package_batches pb ON pb.id=p.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		WHERE p.phone=$1`, phone,
	).Scan(&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone, &pt.Email,
		&pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
		&pt.CreatedAt, &pt.UpdatedAt, &pt.BatchDepartureDate, &pt.PackageName)
	if err != nil {
		return nil, err
	}
	return &pt, nil
}

func (r *participantRepo) List(ctx context.Context, f participant.Filter) ([]participant.Participant, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	i := 1

	if f.BatchID != nil {
		where += fmt.Sprintf(" AND p.batch_id=$%d", i)
		args = append(args, *f.BatchID)
		i++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND p.is_active=$%d", i)
		args = append(args, *f.IsActive)
		i++
	}
	if f.Search != nil {
		where += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.phone ILIKE $%d)", i, i+1)
		pat := "%" + *f.Search + "%"
		args = append(args, pat, pat)
		i += 2
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM participants p
		 JOIN package_batches pb ON pb.id=p.batch_id
		 JOIN packages pkg ON pkg.id=pb.package_id `+where, args...).Scan(&total); err != nil {
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
		SELECT `+participantCols+`
		FROM participants p
		JOIN package_batches pb ON pb.id=p.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		%s ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d`, where, i, i+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []participant.Participant
	for rows.Next() {
		var pt participant.Participant
		if err := rows.Scan(&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone,
			&pt.Email, &pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
			&pt.CreatedAt, &pt.UpdatedAt, &pt.BatchDepartureDate, &pt.PackageName); err != nil {
			return nil, 0, err
		}
		list = append(list, pt)
	}
	return list, total, rows.Err()
}

func (r *participantRepo) Activate(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE participants SET is_active=true,updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *participantRepo) ListByBatch(ctx context.Context, batchID string) ([]participant.Participant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+participantCols+`
		FROM participants p
		JOIN package_batches pb ON pb.id=p.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		WHERE p.batch_id=$1 ORDER BY p.name`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []participant.Participant
	for rows.Next() {
		var pt participant.Participant
		if err := rows.Scan(&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone,
			&pt.Email, &pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
			&pt.CreatedAt, &pt.UpdatedAt, &pt.BatchDepartureDate, &pt.PackageName); err != nil {
			return nil, err
		}
		list = append(list, pt)
	}
	return list, rows.Err()
}

func (r *participantRepo) ListByDepartureDaysAhead(ctx context.Context, days int) ([]participant.Participant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+participantCols+`
		FROM participants p
		JOIN package_batches pb ON pb.id=p.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		WHERE pb.departure_date = CURRENT_DATE + $1 AND p.is_active=true
		ORDER BY p.name`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []participant.Participant
	for rows.Next() {
		var pt participant.Participant
		if err := rows.Scan(&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone,
			&pt.Email, &pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
			&pt.CreatedAt, &pt.UpdatedAt, &pt.BatchDepartureDate, &pt.PackageName); err != nil {
			return nil, err
		}
		list = append(list, pt)
	}
	return list, rows.Err()
}

func (r *participantRepo) ListWithUnpaidInvoiceDaysOld(ctx context.Context, days int) ([]participant.Participant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT `+participantCols+`
		FROM participants p
		JOIN package_batches pb ON pb.id=p.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		JOIN invoices i ON i.participant_id=p.id
		WHERE i.status IN ('diterbitkan','menunggu_bayar')
		AND i.created_at < NOW() - ($1 || ' days')::interval
		ORDER BY p.name`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []participant.Participant
	for rows.Next() {
		var pt participant.Participant
		if err := rows.Scan(&pt.ID, &pt.LeadID, &pt.BatchID, &pt.Name, &pt.Phone,
			&pt.Email, &pt.RoomType, &pt.PortalPassword, &pt.IsActive, &pt.BriefingViewed,
			&pt.CreatedAt, &pt.UpdatedAt, &pt.BatchDepartureDate, &pt.PackageName); err != nil {
			return nil, err
		}
		list = append(list, pt)
	}
	return list, rows.Err()
}
