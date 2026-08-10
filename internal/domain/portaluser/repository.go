package portaluser

import "context"

// Repository persists portal user identities (v2.0 F1).
type Repository interface {
	Create(ctx context.Context, u *PortalUser) error
	GetByID(ctx context.Context, id string) (*PortalUser, error)
	GetByPhone(ctx context.Context, phone string) (*PortalUser, error)
}
