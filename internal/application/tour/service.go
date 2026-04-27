package toursvc

import (
	"context"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/tour"
)

// TourService orchestrates tour package use cases.
type TourService struct {
	repo tour.Repository
}

// NewTourService creates a new TourService.
func NewTourService(repo tour.Repository) *TourService {
	return &TourService{repo: repo}
}

func (s *TourService) ListPackages(ctx context.Context, f tour.Filter) ([]tour.TourPackage, int, error) {
	return s.repo.List(ctx, f)
}

func (s *TourService) GetPackage(ctx context.Context, slug string) (*tour.PackageDetail, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *TourService) CreatePackage(ctx context.Context, p tour.TourPackage) (string, error) {
	return s.repo.Create(ctx, p)
}

func (s *TourService) UpdatePackage(ctx context.Context, id string, p tour.TourPackage) error {
	return s.repo.Update(ctx, id, p)
}

func (s *TourService) DeletePackage(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TourService) ListDestinations(ctx context.Context) ([]tour.Destination, error) {
	return s.repo.ListDestinations(ctx)
}

func (s *TourService) ListTestimonials(ctx context.Context, page, perPage int) ([]tour.Testimonial, error) {
	return s.repo.ListTestimonials(ctx, page, perPage)
}

func (s *TourService) ListGallery(ctx context.Context, packageID string) ([]tour.GalleryImage, error) {
	return s.repo.ListGallery(ctx, packageID)
}

func (s *TourService) AddGalleryImage(ctx context.Context, packageID, imageURL string, caption *string, sortOrder int) (string, error) {
	return s.repo.AddGalleryImage(ctx, packageID, imageURL, caption, sortOrder)
}

func (s *TourService) DeleteGalleryImage(ctx context.Context, packageID, imageID string) error {
	return s.repo.DeleteGalleryImage(ctx, packageID, imageID)
}

func (s *TourService) ListItinerary(ctx context.Context, packageID string) ([]tour.ItineraryItem, error) {
	return s.repo.ListItinerary(ctx, packageID)
}

func (s *TourService) AddItineraryItem(ctx context.Context, packageID string, item tour.ItineraryItem) (string, error) {
	return s.repo.AddItineraryItem(ctx, packageID, item)
}

func (s *TourService) UpdateItineraryItem(ctx context.Context, packageID, itemID string, item tour.ItineraryItem) error {
	return s.repo.UpdateItineraryItem(ctx, packageID, itemID, item)
}

func (s *TourService) DeleteItineraryItem(ctx context.Context, packageID, itemID string) error {
	return s.repo.DeleteItineraryItem(ctx, packageID, itemID)
}
