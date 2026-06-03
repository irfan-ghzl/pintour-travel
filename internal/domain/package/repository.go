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
	List(ctx context.Context, f BatchFilter) ([]PackageBatch, error)
	ListByPackage(ctx context.Context, packageID string) ([]PackageBatch, error)
}
