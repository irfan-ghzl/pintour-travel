package quotation

// Quotation is a price proposal created for a customer.
type Quotation struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	CustomerName  string  `json:"customer_name"`
	CustomerEmail *string `json:"customer_email,omitempty"`
	CustomerPhone *string `json:"customer_phone,omitempty"`
	ValidUntil    *string `json:"valid_until,omitempty"`
	TotalPrice    float64 `json:"total_price"`
	Notes         *string `json:"notes,omitempty"`
	Status        string  `json:"status"`
	PDFURL        *string `json:"pdf_url,omitempty"`
	CreatedAt     string  `json:"created_at"`
	CreatedByName string  `json:"created_by_name"`
}

// Item is a single line entry in a quotation.
type Item struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Category    *string `json:"category,omitempty"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

// Detail is the full quotation view including its line items.
type Detail struct {
	Quotation
	Items []Item `json:"items"`
}

// Filter holds query parameters for listing quotations.
type Filter struct {
	Page    int
	PerPage int
}

// CreateParams carries the data required to create a new quotation.
type CreateParams struct {
	InquiryID     *string
	CreatedBy     *string
	Title         string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	ValidUntil    *string
	TotalPrice    float64
	Notes         string
	Items         []CreateItemParams
}

// CreateItemParams holds data for a single line item.
type CreateItemParams struct {
	Description string
	Category    string
	Quantity    int
	UnitPrice   float64
}
