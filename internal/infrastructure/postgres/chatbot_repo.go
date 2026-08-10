package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
)

type chatbotRepo struct{ db *sql.DB }

func NewChatbotRepo(db *sql.DB) chatbot.Repository { return &chatbotRepo{db} }

func (r *chatbotRepo) Create(ctx context.Context, l *chatbot.Log) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO chatbot_logs (id, phone, role, message, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		RETURNING id, created_at`,
		l.Phone, l.Role, l.Message,
	).Scan(&l.ID, &l.CreatedAt)
}

func (r *chatbotRepo) RecentByPhone(ctx context.Context, phone string, limit int) ([]chatbot.Log, error) {
	if limit <= 0 {
		limit = 10
	}
	// Take the most recent `limit`, then reverse to chronological order.
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, phone, role, message, created_at FROM (
			SELECT id, phone, role, message, created_at
			FROM chatbot_logs WHERE phone=$1
			ORDER BY created_at DESC LIMIT $2
		) t ORDER BY created_at ASC`, phone, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogs(rows)
}

func (r *chatbotRepo) ListByPhone(ctx context.Context, phone string) ([]chatbot.Log, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, phone, role, message, created_at
		FROM chatbot_logs WHERE phone=$1 ORDER BY created_at ASC`, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogs(rows)
}

func (r *chatbotRepo) ListConversations(ctx context.Context, f chatbot.Filter) ([]chatbot.Conversation, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	i := 1
	if f.Phone != "" {
		where += fmt.Sprintf(" AND c.phone LIKE $%d", i)
		args = append(args, "%"+f.Phone+"%")
		i++
	}
	if f.DateFrom != "" {
		where += fmt.Sprintf(" AND c.created_at >= $%d::date", i)
		args = append(args, f.DateFrom)
		i++
	}
	if f.DateTo != "" {
		where += fmt.Sprintf(" AND c.created_at < ($%d::date + INTERVAL '1 day')", i)
		args = append(args, f.DateTo)
		i++
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT c.phone) FROM chatbot_logs c `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	offset := (f.Page - 1) * f.Limit
	args = append(args, f.Limit, offset)

	q := fmt.Sprintf(`
		SELECT c.phone, COUNT(*), MIN(c.created_at), MAX(c.created_at),
		  (SELECT l.id FROM leads l WHERE l.phone=c.phone AND l.deleted_at IS NULL
		   ORDER BY l.created_at DESC LIMIT 1)
		FROM chatbot_logs c
		%s
		GROUP BY c.phone
		ORDER BY MAX(c.created_at) DESC
		LIMIT $%d OFFSET $%d`, where, i, i+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []chatbot.Conversation
	for rows.Next() {
		var c chatbot.Conversation
		if err := rows.Scan(&c.Phone, &c.MessageCount, &c.FirstChat, &c.LastChat, &c.LeadID); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func scanLogs(rows *sql.Rows) ([]chatbot.Log, error) {
	var out []chatbot.Log
	for rows.Next() {
		var l chatbot.Log
		if err := rows.Scan(&l.ID, &l.Phone, &l.Role, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
