package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/notification"
)

type notificationRepo struct{ db *sql.DB }

func NewNotificationRepo(db *sql.DB) notification.Repository { return &notificationRepo{db} }

func (r *notificationRepo) Create(ctx context.Context, n *notification.WANotification) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO wa_notifications
		(id,recipient_phone,recipient_name,message_type,message_content,
		 reference_id,reference_type,status,created_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,'pending',NOW())
		RETURNING id,created_at`,
		n.RecipientPhone, n.RecipientName, n.MessageType, n.MessageContent,
		n.ReferenceID, n.ReferenceType,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *notificationRepo) UpdateStatus(ctx context.Context, id, status string, response interface{}) error {
	respJSON, _ := json.Marshal(response)
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE wa_notifications SET status=$1,fonnte_response=$2,sent_at=$3
		WHERE id=$4`, status, respJSON, now, id)
	return err
}

func (r *notificationRepo) ListByReference(ctx context.Context, refID, refType string) ([]notification.WANotification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,recipient_phone,recipient_name,message_type,message_content,
		reference_id,reference_type,status,fonnte_response,sent_at,created_at
		FROM wa_notifications
		WHERE reference_id=$1 AND reference_type=$2
		ORDER BY created_at DESC`, refID, refType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []notification.WANotification
	for rows.Next() {
		var n notification.WANotification
		var respRaw []byte
		if err := rows.Scan(&n.ID, &n.RecipientPhone, &n.RecipientName,
			&n.MessageType, &n.MessageContent, &n.ReferenceID, &n.ReferenceType,
			&n.Status, &respRaw, &n.SentAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		if len(respRaw) > 0 {
			var resp interface{}
			_ = json.Unmarshal(respRaw, &resp)
			n.FonnteResponse = resp
		}
		list = append(list, n)
	}
	return list, rows.Err()
}
