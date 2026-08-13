package pkgsvc

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
)

// Service handles package/catalog business logic.
type Service struct {
	packages domainPkg.Repository
	images   domainPkg.ImageRepository
	batches  domainPkg.BatchRepository
}

func NewService(packages domainPkg.Repository, images domainPkg.ImageRepository, batches domainPkg.BatchRepository) *Service {
	return &Service{packages: packages, images: images, batches: batches}
}

func (s *Service) CreatePackage(ctx context.Context, p *domainPkg.Package) error {
	if p.Slug == "" {
		p.Slug = toSlug(p.Name)
	}
	return s.packages.Create(ctx, p)
}

func (s *Service) UpdatePackage(ctx context.Context, p *domainPkg.Package) error {
	return s.packages.Update(ctx, p)
}

func (s *Service) DeletePackage(ctx context.Context, id string) error {
	return s.packages.Delete(ctx, id)
}

func (s *Service) GetPackage(ctx context.Context, id string) (*domainPkg.Package, error) {
	return s.packages.GetByID(ctx, id)
}

func (s *Service) GetPackageBySlug(ctx context.Context, slug string) (*domainPkg.Package, error) {
	return s.packages.GetBySlug(ctx, slug)
}

func (s *Service) ListPackages(ctx context.Context, f domainPkg.Filter) ([]domainPkg.Package, int, error) {
	return s.packages.List(ctx, f)
}

func (s *Service) AddImage(ctx context.Context, img *domainPkg.PackageImage) error {
	return s.images.Add(ctx, img)
}

func (s *Service) DeleteImage(ctx context.Context, id string) error {
	return s.images.Delete(ctx, id)
}

func (s *Service) ListImages(ctx context.Context, packageID string) ([]domainPkg.PackageImage, error) {
	return s.images.ListByPackage(ctx, packageID)
}

// CreateBatch opens a departure on a package.
//
// The package is loaded so it can vet the batch itself (§14.4
// Package.AddBatch) — which package the departure belongs to is the package's
// business, not the caller's to assert.
func (s *Service) CreateBatch(ctx context.Context, packageID string, b *domainPkg.PackageBatch) error {
	p, err := s.packages.GetByID(ctx, packageID)
	if err != nil {
		return err
	}
	if err := p.AddBatch(b); err != nil {
		return err
	}
	return s.batches.Create(ctx, b)
}

func (s *Service) UpdateBatch(ctx context.Context, b *domainPkg.PackageBatch) error {
	return s.batches.Update(ctx, b)
}

func (s *Service) GetBatch(ctx context.Context, id string) (*domainPkg.PackageBatch, error) {
	return s.batches.GetByID(ctx, id)
}

func (s *Service) ListBatches(ctx context.Context, packageID string) ([]domainPkg.PackageBatch, error) {
	return s.batches.ListByPackage(ctx, packageID)
}

// ListAllBatches lists departures across every package, for the admin pickers
// that must offer a departure without a package being chosen first.
func (s *Service) ListAllBatches(ctx context.Context, f domainPkg.BatchFilter) ([]domainPkg.PackageBatch, int, error) {
	return s.batches.ListAll(ctx, f)
}

var nonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(s string) string {
	s = strings.ToLower(s)
	s = nonAlpha.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 190 {
		s = s[:190]
	}
	return fmt.Sprintf("%s-%d", s, len(s))
}
