package tour

import "context"

// Repository is the persistence abstraction for the tour domain.
// Implementations live in internal/infrastructure/postgres.
type Repository interface {
	List(ctx context.Context, f Filter) ([]TourPackage, int, error)
	GetBySlug(ctx context.Context, slug string) (*PackageDetail, error)
	Create(ctx context.Context, p TourPackage) (string, error)
	Update(ctx context.Context, id string, p TourPackage) error
	Delete(ctx context.Context, id string) error

	ListDestinations(ctx context.Context) ([]Destination, error)
	ListTestimonials(ctx context.Context, page, perPage int) ([]Testimonial, error)

	ListGallery(ctx context.Context, packageID string) ([]GalleryImage, error)
	AddGalleryImage(ctx context.Context, packageID, imageURL string, caption *string, sortOrder int) (string, error)
	DeleteGalleryImage(ctx context.Context, packageID, imageID string) error

	ListItinerary(ctx context.Context, packageID string) ([]ItineraryItem, error)
	AddItineraryItem(ctx context.Context, packageID string, item ItineraryItem) (string, error)
	UpdateItineraryItem(ctx context.Context, packageID, itemID string, item ItineraryItem) error
	DeleteItineraryItem(ctx context.Context, packageID, itemID string) error
}
