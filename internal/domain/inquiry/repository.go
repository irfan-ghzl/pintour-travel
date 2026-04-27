package inquiry

import "context"

// Repository is the persistence abstraction for the inquiry domain.
type Repository interface {
	Create(ctx context.Context, p CreateParams) (id, createdAt string, err error)
	List(ctx context.Context, f Filter) ([]Inquiry, int, error)
	UpdateStatus(ctx context.Context, id, status string) error
	GetPackageTitle(ctx context.Context, packageID string) (string, error)
}
