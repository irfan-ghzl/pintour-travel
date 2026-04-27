package inquiry

// Inquiry represents a customer consultation request (lead).
type Inquiry struct {
	ID            string   `json:"id"`
	FullName      string   `json:"full_name"`
	Email         *string  `json:"email,omitempty"`
	Phone         *string  `json:"phone,omitempty"`
	Destination   *string  `json:"destination,omitempty"`
	NumPeople     int      `json:"num_people"`
	Budget        *float64 `json:"budget,omitempty"`
	DurationDays  *int     `json:"duration_days,omitempty"`
	DepartureDate *string  `json:"departure_date,omitempty"`
	Status        string   `json:"status"`
	WALink        string   `json:"wa_link"`
	CreatedAt     string   `json:"created_at"`
	PackageTitle  string   `json:"package_title"`
}

// CreateParams carries the data required to persist a new inquiry.
type CreateParams struct {
	FullName      string
	Email         string
	Phone         string
	Destination   string
	TourPackageID *string
	NumPeople     int
	Budget        float64
	DurationDays  int
	DepartureDate *string
	Notes         string
	WALink        string
}

// Filter holds query parameters for listing inquiries.
type Filter struct {
	Status  string
	Page    int
	PerPage int
}
