package booking

// Booking is the aggregate root for a confirmed trip booking.
type Booking struct {
	ID            string  `json:"id"`
	TourPackageID *string `json:"tour_package_id,omitempty"`
	QuotationID   *string `json:"quotation_id,omitempty"`
	BookingCode   string  `json:"booking_code"`
	CustomerName  string  `json:"customer_name"`
	CustomerEmail *string `json:"customer_email,omitempty"`
	CustomerPhone *string `json:"customer_phone,omitempty"`
	DepartureDate string  `json:"departure_date"`
	NumPeople     int     `json:"num_people"`
	TotalPrice    float64 `json:"total_price"`
	PaymentStatus string  `json:"payment_status"`
	BookingStatus string  `json:"booking_status"`
	Notes         *string `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at"`
	PackageTitle  *string `json:"package_title,omitempty"`
}

// Participant is a traveller listed on a booking manifest.
type Participant struct {
	ID          string  `json:"id"`
	FullName    string  `json:"full_name"`
	IDType      string  `json:"id_type"`
	IDNumber    string  `json:"id_number"`
	DateOfBirth *string `json:"date_of_birth,omitempty"`
	Phone       *string `json:"phone,omitempty"`
}

// Detail is the full booking view including its participant manifest.
type Detail struct {
	Booking
	Participants []Participant `json:"participants"`
}

// Filter holds query parameters for listing bookings.
type Filter struct {
	PaymentStatus *string
	BookingStatus *string
	Page          int
	PerPage       int
}

// CreateParams carries the data required to create a new booking.
type CreateParams struct {
	TourPackageID *string
	QuotationID   *string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	DepartureDate string
	NumPeople     int
	TotalPrice    float64
	Notes         string
	Participants  []ParticipantParams
}

// ParticipantParams holds data for a single manifest entry.
type ParticipantParams struct {
	FullName    string
	IDType      string
	IDNumber    string
	DateOfBirth *string
	Phone       *string
}
