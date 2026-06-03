package notification

import "context"

type Repository interface {
	Create(ctx context.Context, n *WANotification) error
	UpdateStatus(ctx context.Context, id, status string, response interface{}) error
	ListByReference(ctx context.Context, refID, refType string) ([]WANotification, error)
}
