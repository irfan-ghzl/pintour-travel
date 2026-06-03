package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/airport"
)

type airportRepo struct{ db *sql.DB }

func NewAirportRepo(db *sql.DB) airport.Repository { return &airportRepo{db} }

func (r *airportRepo) InitForBatch(ctx context.Context, batchID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO airport_checklists (id,participant_id,batch_id,updated_at)
		SELECT gen_random_uuid(),p.id,$1,NOW()
		FROM participants p
		WHERE p.batch_id=$1 AND p.is_active=true
		ON CONFLICT (participant_id,batch_id) DO NOTHING`, batchID)
	return err
}

func (r *airportRepo) GetByParticipant(ctx context.Context, participantID, batchID string) (*airport.Checklist, error) {
	var c airport.Checklist
	err := r.db.QueryRowContext(ctx, `
		SELECT ac.id,ac.participant_id,ac.batch_id,
		ac.baggage_checked,ac.baggage_checked_at,
		ac.ticket_distributed,ac.ticket_distributed_at,
		ac.passport_returned,ac.passport_returned_at,
		ac.handled_by,COALESCE(ac.notes,''),ac.updated_at,
		p.name,p.phone,COALESCE(u.name,'')
		FROM airport_checklists ac
		JOIN participants p ON p.id=ac.participant_id
		LEFT JOIN users u ON u.id=ac.handled_by
		WHERE ac.participant_id=$1 AND ac.batch_id=$2`, participantID, batchID,
	).Scan(&c.ID, &c.ParticipantID, &c.BatchID,
		&c.BaggageChecked, &c.BaggageCheckedAt,
		&c.TicketDistributed, &c.TicketDistributedAt,
		&c.PassportReturned, &c.PassportReturnedAt,
		&c.HandledBy, &c.Notes, &c.UpdatedAt,
		&c.ParticipantName, &c.ParticipantPhone, &c.HandledByName)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *airportRepo) ListByBatch(ctx context.Context, f airport.Filter) ([]airport.Checklist, error) {
	q := `SELECT ac.id,ac.participant_id,ac.batch_id,
		ac.baggage_checked,ac.baggage_checked_at,
		ac.ticket_distributed,ac.ticket_distributed_at,
		ac.passport_returned,ac.passport_returned_at,
		ac.handled_by,COALESCE(ac.notes,''),ac.updated_at,
		p.name,p.phone,COALESCE(u.name,'')
		FROM airport_checklists ac
		JOIN participants p ON p.id=ac.participant_id
		LEFT JOIN users u ON u.id=ac.handled_by
		WHERE ac.batch_id=$1`
	args := []interface{}{f.BatchID}
	if f.Status != nil {
		switch *f.Status {
		case "done":
			q += " AND ac.baggage_checked=true AND ac.ticket_distributed=true AND ac.passport_returned=true"
		case "pending":
			q += " AND (ac.baggage_checked=false OR ac.ticket_distributed=false OR ac.passport_returned=false)"
		}
	}
	q += " ORDER BY p.name"
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChecklists(rows)
}

func (r *airportRepo) UpdateBaggage(ctx context.Context, participantID, batchID, handledBy string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE airport_checklists SET baggage_checked=true,baggage_checked_at=$1,
		handled_by=$2,updated_at=NOW()
		WHERE participant_id=$3 AND batch_id=$4`, now, handledBy, participantID, batchID)
	return err
}

func (r *airportRepo) UpdateTicket(ctx context.Context, participantID, batchID, handledBy string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE airport_checklists SET ticket_distributed=true,ticket_distributed_at=$1,
		handled_by=$2,updated_at=NOW()
		WHERE participant_id=$3 AND batch_id=$4`, now, handledBy, participantID, batchID)
	return err
}

func (r *airportRepo) UpdatePassport(ctx context.Context, participantID, batchID, handledBy string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE airport_checklists SET passport_returned=true,passport_returned_at=$1,
		handled_by=$2,updated_at=NOW()
		WHERE participant_id=$3 AND batch_id=$4`, now, handledBy, participantID, batchID)
	return err
}

func (r *airportRepo) GetBatchProgress(ctx context.Context, batchID string) (*airport.BatchProgress, error) {
	var bp airport.BatchProgress
	err := r.db.QueryRowContext(ctx, `
		SELECT $1::text,
		COUNT(*),
		COUNT(*) FILTER(WHERE baggage_checked AND ticket_distributed AND passport_returned),
		COUNT(*) FILTER(WHERE NOT (baggage_checked AND ticket_distributed AND passport_returned))
		FROM airport_checklists WHERE batch_id=$1`, batchID,
	).Scan(&bp.BatchID, &bp.TotalPax, &bp.DoneCount, &bp.PendingCount)
	if err != nil {
		return nil, err
	}
	return &bp, nil
}

func scanChecklists(rows *sql.Rows) ([]airport.Checklist, error) {
	var list []airport.Checklist
	for rows.Next() {
		var c airport.Checklist
		if err := rows.Scan(&c.ID, &c.ParticipantID, &c.BatchID,
			&c.BaggageChecked, &c.BaggageCheckedAt,
			&c.TicketDistributed, &c.TicketDistributedAt,
			&c.PassportReturned, &c.PassportReturnedAt,
			&c.HandledBy, &c.Notes, &c.UpdatedAt,
			&c.ParticipantName, &c.ParticipantPhone, &c.HandledByName); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
