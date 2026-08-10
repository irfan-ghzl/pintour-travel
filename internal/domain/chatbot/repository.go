package chatbot

import "context"

type Repository interface {
	Create(ctx context.Context, l *Log) error
	// RecentByPhone returns the last `limit` messages for a phone, in chronological order.
	RecentByPhone(ctx context.Context, phone string, limit int) ([]Log, error)
	// ListByPhone returns the full conversation for a phone, chronological.
	ListByPhone(ctx context.Context, phone string) ([]Log, error)
	// ListConversations returns per-phone summaries (paginated).
	ListConversations(ctx context.Context, f Filter) ([]Conversation, int, error)
}
