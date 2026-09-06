package pkg

import "context"

type Repository interface {
	Create(ctx context.Context, p *Package) error
	Update(ctx context.Context, p *Package) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Package, error)
	GetBySlug(ctx context.Context, slug string) (*Package, error)
	List(ctx context.Context, f Filter) ([]Package, int, error)
}

type ImageRepository interface {
	Add(ctx context.Context, img *PackageImage) error
	Delete(ctx context.Context, id string) error
	ListByPackage(ctx context.Context, packageID string) ([]PackageImage, error)
}

type BatchRepository interface {
	Create(ctx context.Context, b *PackageBatch) error
	Update(ctx context.Context, b *PackageBatch) error
	GetByID(ctx context.Context, id string) (*PackageBatch, error)
	ListByPackage(ctx context.Context, packageID string) ([]PackageBatch, error)
	// ListAll answers across every package, paginated, nearest departure first,
	// each row carrying its package name and participant count. The per-package
	// listings above cannot serve the admin pickers: three of the four pages that
	// need a departure have no package in hand to ask about.
	ListAll(ctx context.Context, f BatchFilter) ([]PackageBatch, int, error)
}
