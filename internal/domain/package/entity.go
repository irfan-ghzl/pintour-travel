package pkg

import (
	"encoding/json"
	"time"
)

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
	ID            string    `json:"id"`
	PackageID     string    `json:"package_id"`
	DepartureDate time.Time `json:"departure_date" validate:"required"`
	ReturnDate    time.Time `json:"return_date" validate:"required"`
	Quota         int       `json:"quota" validate:"required,gt=0"`
	PriceSingle   float64   `json:"price_single" validate:"required,gt=0"`
	PriceDouble   float64   `json:"price_double" validate:"required,gt=0"`
	PriceTriple   float64   `json:"price_triple" validate:"required,gt=0"`
	Status        string    `json:"status" validate:"omitempty,oneof=tersedia penuh ditutup"` // default tersedia
	TourLeaderID  *string   `json:"tour_leader_id,omitempty"`
	WaGroupLink   string    `json:"wa_group_link"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Joined
	PackageName    string  `json:"package_name,omitempty"`
	TourLeaderName *string `json:"tour_leader_name,omitempty"`
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
