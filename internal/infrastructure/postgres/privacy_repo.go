package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/privacy"
)

type privacyRepo struct{ db *sql.DB }

// NewDeletionRequestRepo stores erasure requests (§25.5) and performs the
// anonymisation they ask for.
func NewDeletionRequestRepo(db *sql.DB) privacy.Repository { return &privacyRepo{db} }

const deletionRequestCols = `
	r.id, r.participant_id, COALESCE(r.reason,''), r.status, r.requested_at,
	r.processed_by, r.processed_at, COALESCE(r.notes,''),
	COALESCE(p.name,''), COALESCE(p.phone,'')`

func scanDeletionRequest(s interface{ Scan(...any) error }, r *privacy.DeletionRequest) error {
	return s.Scan(&r.ID, &r.ParticipantID, &r.Reason, &r.Status, &r.RequestedAt,
		&r.ProcessedBy, &r.ProcessedAt, &r.Notes, &r.ParticipantName, &r.ParticipantPhone)
}

// Create records a request, or returns the one already open for that
// participant. Pressing the button twice is what a worried person does; it must
// not put the same person in the admin queue twice.
func (r *privacyRepo) Create(ctx context.Context, req *privacy.DeletionRequest) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO data_deletion_requests (participant_id, reason, status)
		VALUES ($1, NULLIF($2,''), $3)
		ON CONFLICT (participant_id) WHERE status = 'menunggu' DO NOTHING
		RETURNING id, requested_at, status`,
		req.ParticipantID, req.Reason, privacy.StatusPending,
	).Scan(&req.ID, &req.RequestedAt, &req.Status)

	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	// DO NOTHING returned no row: one is already open. Hand that one back so the
	// caller answers with the request that exists rather than reporting a clash.
	return r.db.QueryRowContext(ctx, `
		SELECT `+deletionRequestCols+`
		FROM data_deletion_requests r
		LEFT JOIN participants p ON p.id = r.participant_id
		WHERE r.participant_id = $1 AND r.status = $2`,
		req.ParticipantID, privacy.StatusPending,
	).Scan(&req.ID, &req.ParticipantID, &req.Reason, &req.Status, &req.RequestedAt,
		&req.ProcessedBy, &req.ProcessedAt, &req.Notes,
		&req.ParticipantName, &req.ParticipantPhone)
}

func (r *privacyRepo) GetByID(ctx context.Context, id string) (*privacy.DeletionRequest, error) {
	var req privacy.DeletionRequest
	err := scanDeletionRequest(r.db.QueryRowContext(ctx, `
		SELECT `+deletionRequestCols+`
		FROM data_deletion_requests r
		LEFT JOIN participants p ON p.id = r.participant_id
		WHERE r.id = $1`, id), &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *privacyRepo) List(ctx context.Context, status string) ([]privacy.DeletionRequest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+deletionRequestCols+`
		FROM data_deletion_requests r
		LEFT JOIN participants p ON p.id = r.participant_id
		WHERE ($1 = '' OR r.status = $1)
		ORDER BY r.requested_at`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []privacy.DeletionRequest{}
	for rows.Next() {
		var req privacy.DeletionRequest
		if err := scanDeletionRequest(rows, &req); err != nil {
			return nil, err
		}
		list = append(list, req)
	}
	return list, rows.Err()
}

// Anonymise overwrites the participant's identifying fields, soft-deletes their
// sensitive documents, and closes the request — in one transaction, so the
// database is never left claiming an erasure that did not happen.
//
// What survives, and why: the participant row itself (§25.4 anonymises rather
// than removes, so departure counts stay whole) and every invoice attached to it
// (a financial record, and §25.5's "kecuali yang wajib dipertahankan secara
// hukum"). What goes: name, phone, email, NIK, and the portal credential — the
// fields that make the row a person.
func (r *privacyRepo) Anonymise(ctx context.Context, requestID, processedBy, notes string) error {
	return r.closeRequest(ctx, requestID, processedBy, notes, privacy.StatusDone,
		func(ctx context.Context, tx *sql.Tx, participantID string) error {
			if _, err := tx.ExecContext(ctx, `
				UPDATE participants SET
					name = 'Peserta Dihapus',
					phone = 'deleted-' || LEFT(id::text, 8),
					email = NULL,
					nik = NULL,
					portal_password = '',
					is_active = false,
					anonymized_at = NOW(),
					deleted_at = NOW(),
					updated_at = NOW()
				WHERE id = $1`, participantID); err != nil {
				return fmt.Errorf("anonymise participant: %w", err)
			}
			// The scans themselves are the sensitive part and are not statistics.
			if _, err := tx.ExecContext(ctx, `
				UPDATE documents SET deleted_at = NOW()
				WHERE participant_id = $1 AND deleted_at IS NULL`, participantID); err != nil {
				return fmt.Errorf("remove documents: %w", err)
			}
			return nil
		})
}

// Reject closes a request without touching any data.
func (r *privacyRepo) Reject(ctx context.Context, requestID, processedBy, notes string) error {
	return r.closeRequest(ctx, requestID, processedBy, notes, privacy.StatusRejected, nil)
}

// closeRequest moves a pending request to a final status, optionally doing work
// against the participant first. The status is changed with the pending state in
// the WHERE clause, so two admins acting on the same queue entry cannot both
// succeed — the second sees ErrAlreadyProcessed instead of anonymising a second
// time or overturning the first decision.
func (r *privacyRepo) closeRequest(
	ctx context.Context,
	requestID, processedBy, notes, status string,
	work func(context.Context, *sql.Tx, string) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	var participantID string
	err = tx.QueryRowContext(ctx, `
		UPDATE data_deletion_requests
		SET status = $1, processed_by = NULLIF($2,'')::uuid, processed_at = NOW(),
		    notes = NULLIF($3,'')
		WHERE id = $4 AND status = $5
		RETURNING participant_id`,
		status, processedBy, notes, requestID, privacy.StatusPending).Scan(&participantID)
	if err == sql.ErrNoRows {
		return privacy.ErrAlreadyProcessed
	}
	if err != nil {
		return err
	}

	if work != nil {
		if err := work(ctx, tx, participantID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
