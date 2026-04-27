package tour

// TourPackage is the core aggregate for a tour product.
type TourPackage struct {
	ID                 string  `json:"id"`
	DestinationID      *string `json:"destination_id,omitempty"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Description        string  `json:"description"`
	Price              float64 `json:"price"`
	PriceLabel         string  `json:"price_label"`
	DurationDays       int     `json:"duration_days"`
	MaxParticipants    *int    `json:"max_participants"`
	MinParticipants    int     `json:"min_participants"`
	PackageType        string  `json:"package_type"`
	CoverImageURL      string  `json:"cover_image_url"`
	IsActive           bool    `json:"is_active"`
	CreatedAt          string  `json:"created_at"`
	DestinationName    *string `json:"destination_name,omitempty"`
	DestinationCountry *string `json:"destination_country,omitempty"`
}

// Destination represents a travel destination.
type Destination struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Country     string  `json:"country"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

// ItineraryItem is a single day/activity entry inside a package.
type ItineraryItem struct {
	ID           string `json:"id"`
	DayNumber    int    `json:"day_number"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Location     string `json:"location"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	ActivityType string `json:"activity_type"`
	SortOrder    int    `json:"sort_order"`
}

// GalleryImage is a photo associated with a package.
type GalleryImage struct {
	ID        string  `json:"id"`
	ImageURL  string  `json:"image_url"`
	Caption   *string `json:"caption,omitempty"`
	SortOrder int     `json:"sort_order"`
}

// Testimonial is a customer review for a package.
type Testimonial struct {
	ID           string  `json:"id"`
	CustomerName string  `json:"customer_name"`
	Content      string  `json:"content"`
	Rating       int     `json:"rating"`
	PhotoURL     *string `json:"photo_url,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// PackageDetail is the full view returned by GetBySlug.
type PackageDetail struct {
	TourPackage
	Itinerary []ItineraryItem `json:"itinerary"`
}

// Filter holds query parameters for listing packages.
type Filter struct {
	DestinationID *string
	PackageType   *string
	PriceMin      *float64
	PriceMax      *float64
	DurationDays  *int
	Page          int
	PerPage       int
}
