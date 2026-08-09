package pkg

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
)

// ErrBatchPackageMismatch is returned when a departure already belonging to one
// package is offered to another.
var ErrBatchPackageMismatch = errors.New("batch sudah terikat ke paket lain")

// OpenBatchStatus is the batch status that still takes bookings. Keep in sync
// with the package_batches_status_check constraint in
// db/migrations/003_prd_schema.sql.
const OpenBatchStatus = "tersedia"

// Package is the core product aggregate.
//
// The validate tags apply when a CMS request binds into this type (PRD §19.3);
// each bound mirrors a constraint on the packages table in
// db/migrations/003_prd_schema.sql. Slug is not required because
// Service.CreatePackage derives it from the name when it is blank.
type Package struct {
	ID              string          `json:"id"`
	Name            string          `json:"name" validate:"required"`
	Slug            string          `json:"slug"`
	Destination     string          `json:"destination" validate:"required"`
	Category        string          `json:"category" validate:"omitempty,oneof=reguler umroh halal honeymoon"` // default reguler
	DurationDays    int             `json:"duration_days" validate:"required,gte=1,lte=365"`
	Description     string          `json:"description"`
	Itinerary       json.RawMessage `json:"itinerary"`    // []ItineraryDay
	Requirements    json.RawMessage `json:"requirements"` // []string
	Facilities      json.RawMessage `json:"facilities"`   // FR-CAT-03
	TermsConditions string          `json:"terms_conditions"`
	VisaInfo        string          `json:"visa_info"`
	BasePrice       float64         `json:"base_price" validate:"required,gt=0"`
	IsActive        bool            `json:"is_active"`
	Thumbnail       string          `json:"thumbnail"`
	CreatedBy       string          `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at,omitempty"`
}

// Activate publishes the package to the catalogue (§14.4).
func (p *Package) Activate() { p.IsActive = true }

// Deactivate withdraws the package from the catalogue (§14.4). It is not a
// delete: the package keeps its batches, invoices and history, it just stops
// being offered.
func (p *Package) Deactivate() { p.IsActive = false }

// AddBatch links a departure to this package (§14.4).
//
// The rows themselves are written by the repository — a package holds no slice
// of batches, per the §14.4 note that one-to-many is realised as a repository
// query. What belongs here is the part that needs no database: the batch has to
// be this package's, and a batch already claimed by another package is refused
// rather than quietly re-pointed.
func (p *Package) AddBatch(b *PackageBatch) error {
	if b == nil {
		return errors.New("batch tidak boleh kosong")
	}
	if p == nil || p.ID == "" {
		return errors.New("paket belum punya id")
	}
	if b.PackageID != "" && b.PackageID != p.ID {
		return ErrBatchPackageMismatch
	}
	b.PackageID = p.ID
	return nil
}

// ItineraryDay represents one day in a package itinerary.
type ItineraryDay struct {
	Day         int      `json:"day"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Activities  []string `json:"activities"`
}

// PackageImage is a photo associated with a package.
type PackageImage struct {
	ID          string    `json:"id"`
	PackageID   string    `json:"package_id"`
	FilePath    string    `json:"file_path" validate:"required"`
	AltText     string    `json:"alt_text"`
	SortOrder   int       `json:"sort_order"`
	IsThumbnail bool      `json:"is_thumbnail"`
	CreatedAt   time.Time `json:"created_at"`
}

// PackageBatch is a scheduled departure for a package.
type PackageBatch struct {
	ID            string        `json:"id"`
	PackageID     string        `json:"package_id"`
	DepartureDate calendar.Date `json:"departure_date" validate:"required"`
	ReturnDate    calendar.Date `json:"return_date" validate:"required"`
	Quota         int           `json:"quota" validate:"required,gt=0"`
	PriceSingle   float64       `json:"price_single" validate:"required,gt=0"`
	PriceDouble   float64       `json:"price_double" validate:"required,gt=0"`
	PriceTriple   float64       `json:"price_triple" validate:"required,gt=0"`
	Status        string        `json:"status" validate:"omitempty,oneof=tersedia penuh ditutup"` // default tersedia
	TourLeaderID  *string       `json:"tour_leader_id,omitempty"`
	WaGroupLink   string        `json:"wa_group_link"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	// Joined
	PackageName    string  `json:"package_name,omitempty"`
	TourLeaderName *string `json:"tour_leader_name,omitempty"`
}

// HasAvailableSeats reports whether the batch can still take a booking (§14.4).
//
// sold is passed in because the filled count is not a column — §14.4's own note
// says so — and counting participants is a query, which is exactly the kind of
// work a domain method must not do. The caller that counted them hands the
// number over.
func (b *PackageBatch) HasAvailableSeats(sold int) bool {
	if b == nil {
		return false
	}
	if b.Status != "" && b.Status != OpenBatchStatus {
		return false
	}
	return b.SeatsRemaining(sold) > 0
}

// SeatsRemaining is the seat count left in the batch, never below zero — an
// oversold batch has no seats, not negative ones.
func (b *PackageBatch) SeatsRemaining(sold int) int {
	if b == nil || b.Quota <= 0 {
		return 0
	}
	if remaining := b.Quota - sold; remaining > 0 {
		return remaining
	}
	return 0
}

// DaysUntilDeparture is the whole days left before the batch leaves (§14.4).
// Negative once it has left, zero on the day itself.
func (b *PackageBatch) DaysUntilDeparture() int {
	return b.DaysUntilDepartureFrom(time.Now())
}

// DaysUntilDepartureFrom is DaysUntilDeparture measured from a given moment,
// which is what makes the countdown testable without waiting for a calendar.
func (b *PackageBatch) DaysUntilDepartureFrom(now time.Time) int {
	if b == nil || b.DepartureDate.IsZero() {
		return 0
	}
	// Whole calendar days, so the number matches what a person counting on a
	// calendar would say rather than depending on the time of day it is asked.
	return calendar.Of(now).DaysUntil(b.DepartureDate)
}

// Filter holds query parameters for listing packages.
type Filter struct {
	Category       *string
	Destination    *string
	PriceMin       *float64
	PriceMax       *float64
	DurationMin    *int
	DurationMax    *int
	IsActive       *bool
	DepartureMonth *string
	Page           int
	PerPage        int
}

// BatchFilter holds filter params for listing batches.
type BatchFilter struct {
	PackageID *string
	Status    *string
}
