package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// TourHandler handles HTTP requests for tour packages and related resources.
type TourHandler struct {
	db *sql.DB
}

// NewTourHandler creates a new TourHandler.
func NewTourHandler(db *sql.DB) *TourHandler {
	return &TourHandler{db: db}
}

// ListPackages godoc
//
//	@Summary     List tour packages
//	@Description Returns a paginated list of active tour packages
//	@Tags        packages
//	@Produce     json
//	@Param       page     query int false "Page number (default 1)"
//	@Param       per_page query int false "Items per page (default 10)"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/packages [get]
func (h *TourHandler) ListPackages(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 10)
	if perPage > 50 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	rows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT tp.id, tp.title, tp.slug, tp.description, tp.price, tp.price_label,
		       tp.duration_days, tp.max_participants, tp.min_participants,
		       tp.package_type, tp.cover_image_url, tp.is_active, tp.created_at,
		       d.name AS destination_name, d.country AS destination_country
		FROM tour_packages tp
		LEFT JOIN destinations d ON d.id = tp.destination_id
		WHERE tp.is_active = true
		ORDER BY tp.created_at DESC
		LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch packages")
	}
	defer rows.Close()

	var packages []map[string]interface{}
	for rows.Next() {
		var (
			id, title, slug, description, priceLabel, packageType, coverImageURL string
			price                                                                  float64
			durationDays, minParticipants                                          int
			maxParticipants                                                         *int
			isActive                                                                bool
			createdAt                                                               string
			destName, destCountry                                                   *string
		)
		if err := rows.Scan(
			&id, &title, &slug, &description, &price, &priceLabel,
			&durationDays, &maxParticipants, &minParticipants,
			&packageType, &coverImageURL, &isActive, &createdAt,
			&destName, &destCountry,
		); err != nil {
			continue
		}
		pkg := map[string]interface{}{
			"id": id, "title": title, "slug": slug, "description": description,
			"price": price, "price_label": priceLabel,
			"duration_days": durationDays, "max_participants": maxParticipants,
			"min_participants": minParticipants, "package_type": packageType,
			"cover_image_url": coverImageURL, "is_active": isActive,
			"created_at":          createdAt,
			"destination_name":    destName,
			"destination_country": destCountry,
		}
		packages = append(packages, pkg)
	}
	if packages == nil {
		packages = []map[string]interface{}{}
	}

	var total int
	h.db.QueryRowContext(c.Request().Context(),
		`SELECT COUNT(*) FROM tour_packages WHERE is_active = true`).Scan(&total)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":     packages,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetPackage godoc
//
//	@Summary     Get a tour package
//	@Description Returns a single tour package with its itinerary
//	@Tags        packages
//	@Produce     json
//	@Param       slug path string true "Package slug"
//	@Success     200 {object} map[string]interface{}
//	@Failure     404 {object} map[string]interface{}
//	@Router      /api/v1/packages/{slug} [get]
func (h *TourHandler) GetPackage(c echo.Context) error {
	slug := c.Param("slug")

	var pkg map[string]interface{}
	row := h.db.QueryRowContext(c.Request().Context(), `
		SELECT tp.id, tp.title, tp.slug, tp.description, tp.price, tp.price_label,
		       tp.duration_days, tp.max_participants, tp.min_participants,
		       tp.package_type, tp.cover_image_url, tp.is_active, tp.created_at,
		       COALESCE(d.name, '') AS destination_name,
		       COALESCE(d.country, '') AS destination_country
		FROM tour_packages tp
		LEFT JOIN destinations d ON d.id = tp.destination_id
		WHERE tp.slug = $1`, slug)

	var (
		id, title, dbSlug, description, priceLabel, packageType, coverImageURL string
		price                                                                    float64
		durationDays, minParticipants                                            int
		maxParticipants                                                           *int
		isActive                                                                  bool
		createdAt, destName, destCountry                                          string
	)
	if err := row.Scan(
		&id, &title, &dbSlug, &description, &price, &priceLabel,
		&durationDays, &maxParticipants, &minParticipants,
		&packageType, &coverImageURL, &isActive, &createdAt,
		&destName, &destCountry,
	); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "package not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch package")
	}

	pkg = map[string]interface{}{
		"id": id, "title": title, "slug": dbSlug, "description": description,
		"price": price, "price_label": priceLabel,
		"duration_days": durationDays, "max_participants": maxParticipants,
		"min_participants": minParticipants, "package_type": packageType,
		"cover_image_url": coverImageURL, "is_active": isActive, "created_at": createdAt,
		"destination_name":    destName,
		"destination_country": destCountry,
	}

	itineraryRows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT id, day_number, title, description, location, start_time, end_time,
		       activity_type, sort_order
		FROM itinerary_items
		WHERE tour_package_id = $1
		ORDER BY day_number ASC, sort_order ASC`, id)
	if err == nil {
		defer itineraryRows.Close()
		var items []map[string]interface{}
		for itineraryRows.Next() {
			var (
				itemID, itemTitle, itemDesc, itemLoc, startTime, endTime, actType string
				dayNumber, sortOrder                                                int
			)
			if err := itineraryRows.Scan(&itemID, &dayNumber, &itemTitle, &itemDesc,
				&itemLoc, &startTime, &endTime, &actType, &sortOrder); err != nil {
				continue
			}
			items = append(items, map[string]interface{}{
				"id": itemID, "day_number": dayNumber, "title": itemTitle,
				"description": itemDesc, "location": itemLoc,
				"start_time": startTime, "end_time": endTime,
				"activity_type": actType, "sort_order": sortOrder,
			})
		}
		if items == nil {
			items = []map[string]interface{}{}
		}
		pkg["itinerary"] = items
	}

	return c.JSON(http.StatusOK, pkg)
}

// CreatePackage godoc
//
//	@Summary     Create a tour package (admin)
//	@Tags        packages
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/packages [post]
func (h *TourHandler) CreatePackage(c echo.Context) error {
	var body struct {
		DestinationID   *string  `json:"destination_id"`
		Title           string   `json:"title"`
		Slug            string   `json:"slug"`
		Description     string   `json:"description"`
		Price           float64  `json:"price"`
		PriceLabel      string   `json:"price_label"`
		DurationDays    int      `json:"duration_days"`
		MaxParticipants *int     `json:"max_participants"`
		MinParticipants int      `json:"min_participants"`
		PackageType     string   `json:"package_type"`
		CoverImageURL   string   `json:"cover_image_url"`
		IsActive        bool     `json:"is_active"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if body.Title == "" || body.Slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and slug are required")
	}
	if body.PackageType == "" {
		body.PackageType = "regular"
	}
	if body.MinParticipants == 0 {
		body.MinParticipants = 1
	}

	var id string
	err := h.db.QueryRowContext(c.Request().Context(), `
		INSERT INTO tour_packages
		  (destination_id, title, slug, description, price, price_label,
		   duration_days, max_participants, min_participants, package_type,
		   cover_image_url, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id`,
		body.DestinationID, body.Title, body.Slug, body.Description,
		body.Price, body.PriceLabel, body.DurationDays,
		body.MaxParticipants, body.MinParticipants,
		body.PackageType, body.CoverImageURL, body.IsActive,
	).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create package: "+err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id, "slug": body.Slug})
}

// UpdatePackage godoc
//
//	@Summary     Update a tour package (admin)
//	@Tags        packages
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Package ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/packages/{id} [put]
func (h *TourHandler) UpdatePackage(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		DestinationID   *string  `json:"destination_id"`
		Title           string   `json:"title"`
		Slug            string   `json:"slug"`
		Description     string   `json:"description"`
		Price           float64  `json:"price"`
		PriceLabel      string   `json:"price_label"`
		DurationDays    int      `json:"duration_days"`
		MaxParticipants *int     `json:"max_participants"`
		MinParticipants int      `json:"min_participants"`
		PackageType     string   `json:"package_type"`
		CoverImageURL   string   `json:"cover_image_url"`
		IsActive        bool     `json:"is_active"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	_, err := h.db.ExecContext(c.Request().Context(), `
		UPDATE tour_packages SET
		  destination_id=$2, title=$3, slug=$4, description=$5, price=$6,
		  price_label=$7, duration_days=$8, max_participants=$9,
		  min_participants=$10, package_type=$11, cover_image_url=$12,
		  is_active=$13, updated_at=NOW()
		WHERE id=$1`,
		id, body.DestinationID, body.Title, body.Slug, body.Description,
		body.Price, body.PriceLabel, body.DurationDays,
		body.MaxParticipants, body.MinParticipants,
		body.PackageType, body.CoverImageURL, body.IsActive,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update package")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id})
}

// DeletePackage godoc
//
//	@Summary     Delete a tour package (admin)
//	@Tags        packages
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Package ID"
//	@Success     204
//	@Router      /api/v1/admin/packages/{id} [delete]
func (h *TourHandler) DeletePackage(c echo.Context) error {
	id := c.Param("id")
	if _, err := h.db.ExecContext(c.Request().Context(),
		`DELETE FROM tour_packages WHERE id=$1`, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete package")
	}
	return c.NoContent(http.StatusNoContent)
}

// ListDestinations godoc
//
//	@Summary     List destinations
//	@Tags        destinations
//	@Produce     json
//	@Success     200 {array} map[string]interface{}
//	@Router      /api/v1/destinations [get]
func (h *TourHandler) ListDestinations(c echo.Context) error {
	rows, err := h.db.QueryContext(c.Request().Context(),
		`SELECT id, name, country, description, image_url FROM destinations ORDER BY name ASC`)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch destinations")
	}
	defer rows.Close()

	var destinations []map[string]interface{}
	for rows.Next() {
		var id, name, country string
		var description, imageURL *string
		if err := rows.Scan(&id, &name, &country, &description, &imageURL); err != nil {
			continue
		}
		destinations = append(destinations, map[string]interface{}{
			"id": id, "name": name, "country": country,
			"description": description, "image_url": imageURL,
		})
	}
	if destinations == nil {
		destinations = []map[string]interface{}{}
	}
	return c.JSON(http.StatusOK, destinations)
}

// ListTestimonials godoc
//
//	@Summary     List published testimonials
//	@Tags        testimonials
//	@Produce     json
//	@Success     200 {array} map[string]interface{}
//	@Router      /api/v1/testimonials [get]
func (h *TourHandler) ListTestimonials(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 10)
	offset := (page - 1) * perPage

	rows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT id, customer_name, content, rating, photo_url, created_at
		FROM testimonials
		WHERE is_published = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch testimonials")
	}
	defer rows.Close()

	var testimonials []map[string]interface{}
	for rows.Next() {
		var id, customerName, content, createdAt string
		var rating int
		var photoURL *string
		if err := rows.Scan(&id, &customerName, &content, &rating, &photoURL, &createdAt); err != nil {
			continue
		}
		testimonials = append(testimonials, map[string]interface{}{
			"id": id, "customer_name": customerName, "content": content,
			"rating": rating, "photo_url": photoURL, "created_at": createdAt,
		})
	}
	if testimonials == nil {
		testimonials = []map[string]interface{}{}
	}
	return c.JSON(http.StatusOK, testimonials)
}

// queryInt reads an integer query parameter with a fallback default.
func queryInt(c echo.Context, name string, defaultVal int) int {
	s := c.QueryParam(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}
