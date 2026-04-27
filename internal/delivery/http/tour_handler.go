package httpdelivery

import (
	"database/sql"
	"net/http"
	"strings"

	toursvc "github.com/irfan-ghzl/pintour-travel/internal/application/tour"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/tour"
	"github.com/labstack/echo/v4"
)

// TourHandler handles HTTP requests for tour packages and related resources.
type TourHandler struct {
	svc *toursvc.TourService
}

// NewTourHandler creates a new TourHandler.
func NewTourHandler(svc *toursvc.TourService) *TourHandler {
	return &TourHandler{svc: svc}
}

// ListPackages godoc
//
//	@Summary     List tour packages
//	@Tags        packages
//	@Produce     json
//	@Param       page           query int    false "Page (default 1)"
//	@Param       per_page       query int    false "Items per page (default 10)"
//	@Param       destination_id query string false "Filter by destination UUID"
//	@Param       package_type   query string false "Filter by type (regular|premium|honeymoon)"
//	@Param       price_min      query number false "Minimum price"
//	@Param       price_max      query number false "Maximum price"
//	@Param       duration_days  query int    false "Exact duration in days"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/packages [get]
func (h *TourHandler) ListPackages(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 10)
	if perPage > 50 {
		perPage = 50
	}

	f := tour.Filter{
		DestinationID: queryStringPtr(c, "destination_id"),
		PackageType:   queryStringPtr(c, "package_type"),
		PriceMin:      queryFloat64Ptr(c, "price_min"),
		PriceMax:      queryFloat64Ptr(c, "price_max"),
		DurationDays:  queryIntPtr(c, "duration_days"),
		Page:          page,
		PerPage:       perPage,
	}

	packages, total, err := h.svc.ListPackages(c.Request().Context(), f)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch packages")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": packages, "total": total, "page": page, "per_page": perPage,
	})
}

// GetPackage godoc
//
//	@Summary     Get a tour package by slug
//	@Tags        packages
//	@Produce     json
//	@Param       slug path string true "Package slug"
//	@Success     200 {object} tour.PackageDetail
//	@Failure     404 {object} map[string]interface{}
//	@Router      /api/v1/packages/{slug} [get]
func (h *TourHandler) GetPackage(c echo.Context) error {
	detail, err := h.svc.GetPackage(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch package")
	}
	if detail == nil {
		return echo.NewHTTPError(http.StatusNotFound, "package not found")
	}
	return c.JSON(http.StatusOK, detail)
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
		DestinationID   *string `json:"destination_id"`
		Title           string  `json:"title"`
		Slug            string  `json:"slug"`
		Description     string  `json:"description"`
		Price           float64 `json:"price"`
		PriceLabel      string  `json:"price_label"`
		DurationDays    int     `json:"duration_days"`
		MaxParticipants *int    `json:"max_participants"`
		MinParticipants int     `json:"min_participants"`
		PackageType     string  `json:"package_type"`
		CoverImageURL   string  `json:"cover_image_url"`
		IsActive        bool    `json:"is_active"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Slug) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and slug are required")
	}
	if body.PackageType == "" {
		body.PackageType = "regular"
	}
	if body.MinParticipants == 0 {
		body.MinParticipants = 1
	}

	p := tour.TourPackage{
		DestinationID:   body.DestinationID,
		Title:           body.Title,
		Slug:            body.Slug,
		Description:     body.Description,
		Price:           body.Price,
		PriceLabel:      body.PriceLabel,
		DurationDays:    body.DurationDays,
		MaxParticipants: body.MaxParticipants,
		MinParticipants: body.MinParticipants,
		PackageType:     body.PackageType,
		CoverImageURL:   body.CoverImageURL,
		IsActive:        body.IsActive,
	}

	id, err := h.svc.CreatePackage(c.Request().Context(), p)
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
		DestinationID   *string `json:"destination_id"`
		Title           string  `json:"title"`
		Slug            string  `json:"slug"`
		Description     string  `json:"description"`
		Price           float64 `json:"price"`
		PriceLabel      string  `json:"price_label"`
		DurationDays    int     `json:"duration_days"`
		MaxParticipants *int    `json:"max_participants"`
		MinParticipants int     `json:"min_participants"`
		PackageType     string  `json:"package_type"`
		CoverImageURL   string  `json:"cover_image_url"`
		IsActive        bool    `json:"is_active"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	p := tour.TourPackage{
		DestinationID:   body.DestinationID,
		Title:           body.Title,
		Slug:            body.Slug,
		Description:     body.Description,
		Price:           body.Price,
		PriceLabel:      body.PriceLabel,
		DurationDays:    body.DurationDays,
		MaxParticipants: body.MaxParticipants,
		MinParticipants: body.MinParticipants,
		PackageType:     body.PackageType,
		CoverImageURL:   body.CoverImageURL,
		IsActive:        body.IsActive,
	}

	if err := h.svc.UpdatePackage(c.Request().Context(), id, p); err != nil {
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
	if err := h.svc.DeletePackage(c.Request().Context(), c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete package")
	}
	return c.NoContent(http.StatusNoContent)
}

// ListDestinations godoc
//
//	@Summary     List destinations
//	@Tags        destinations
//	@Produce     json
//	@Success     200 {array} tour.Destination
//	@Router      /api/v1/destinations [get]
func (h *TourHandler) ListDestinations(c echo.Context) error {
	destinations, err := h.svc.ListDestinations(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch destinations")
	}
	return c.JSON(http.StatusOK, destinations)
}

// ListTestimonials godoc
//
//	@Summary     List published testimonials
//	@Tags        testimonials
//	@Produce     json
//	@Success     200 {array} tour.Testimonial
//	@Router      /api/v1/testimonials [get]
func (h *TourHandler) ListTestimonials(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 10)
	testimonials, err := h.svc.ListTestimonials(c.Request().Context(), page, perPage)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch testimonials")
	}
	return c.JSON(http.StatusOK, testimonials)
}

// ListGallery godoc
//
//	@Summary     List gallery images for a package
//	@Tags        gallery
//	@Produce     json
//	@Param       package_id path string true "Tour Package ID"
//	@Success     200 {array} tour.GalleryImage
//	@Router      /api/v1/packages/{package_id}/gallery [get]
func (h *TourHandler) ListGallery(c echo.Context) error {
	gallery, err := h.svc.ListGallery(c.Request().Context(), c.Param("package_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch gallery")
	}
	return c.JSON(http.StatusOK, gallery)
}

// AddGalleryImage godoc
//
//	@Summary     Add an image to a package gallery (admin)
//	@Tags        gallery
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/packages/{package_id}/gallery [post]
func (h *TourHandler) AddGalleryImage(c echo.Context) error {
	packageID := c.Param("package_id")
	var body struct {
		ImageURL  string `json:"image_url"`
		Caption   string `json:"caption"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if body.ImageURL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "image_url is required")
	}

	var caption *string
	if body.Caption != "" {
		caption = &body.Caption
	}

	id, err := h.svc.AddGalleryImage(c.Request().Context(), packageID, body.ImageURL, caption, body.SortOrder)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add image")
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id": id, "image_url": body.ImageURL, "caption": caption,
	})
}

// DeleteGalleryImage godoc
//
//	@Summary     Delete a gallery image (admin)
//	@Tags        gallery
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Param       image_id   path string true "Image ID"
//	@Success     204
//	@Router      /api/v1/admin/packages/{package_id}/gallery/{image_id} [delete]
func (h *TourHandler) DeleteGalleryImage(c echo.Context) error {
	err := h.svc.DeleteGalleryImage(c.Request().Context(), c.Param("package_id"), c.Param("image_id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "image not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete image")
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Itinerary ──────────────────────────────────────────────────────────────

// ListItinerary godoc
//
//	@Summary     List itinerary items for a package (admin)
//	@Tags        itinerary
//	@Produce     json
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Success     200 {array} tour.ItineraryItem
//	@Router      /api/v1/admin/packages/{package_id}/itinerary [get]
func (h *TourHandler) ListItinerary(c echo.Context) error {
	items, err := h.svc.ListItinerary(c.Request().Context(), c.Param("package_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch itinerary")
	}
	return c.JSON(http.StatusOK, items)
}

// AddItineraryItem godoc
//
//	@Summary     Add an itinerary item to a package (admin)
//	@Tags        itinerary
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/packages/{package_id}/itinerary [post]
func (h *TourHandler) AddItineraryItem(c echo.Context) error {
	packageID := c.Param("package_id")
	var body struct {
		DayNumber    int    `json:"day_number"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Location     string `json:"location"`
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		ActivityType string `json:"activity_type"`
		SortOrder    int    `json:"sort_order"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(body.Title) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if body.DayNumber < 1 {
		body.DayNumber = 1
	}

	id, err := h.svc.AddItineraryItem(c.Request().Context(), packageID, tour.ItineraryItem{
		DayNumber:    body.DayNumber,
		Title:        body.Title,
		Description:  body.Description,
		Location:     body.Location,
		StartTime:    body.StartTime,
		EndTime:      body.EndTime,
		ActivityType: body.ActivityType,
		SortOrder:    body.SortOrder,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add itinerary item: "+err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id})
}

// UpdateItineraryItem godoc
//
//	@Summary     Update an itinerary item (admin)
//	@Tags        itinerary
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Param       item_id    path string true "Itinerary Item ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/packages/{package_id}/itinerary/{item_id} [put]
func (h *TourHandler) UpdateItineraryItem(c echo.Context) error {
	packageID := c.Param("package_id")
	itemID := c.Param("item_id")
	var body struct {
		DayNumber    int    `json:"day_number"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Location     string `json:"location"`
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		ActivityType string `json:"activity_type"`
		SortOrder    int    `json:"sort_order"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(body.Title) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if body.DayNumber < 1 {
		body.DayNumber = 1
	}

	err := h.svc.UpdateItineraryItem(c.Request().Context(), packageID, itemID, tour.ItineraryItem{
		DayNumber:    body.DayNumber,
		Title:        body.Title,
		Description:  body.Description,
		Location:     body.Location,
		StartTime:    body.StartTime,
		EndTime:      body.EndTime,
		ActivityType: body.ActivityType,
		SortOrder:    body.SortOrder,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "itinerary item not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update itinerary item")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": itemID, "package_id": packageID})
}

// DeleteItineraryItem godoc
//
//	@Summary     Delete an itinerary item (admin)
//	@Tags        itinerary
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Param       item_id    path string true "Itinerary Item ID"
//	@Success     204
//	@Router      /api/v1/admin/packages/{package_id}/itinerary/{item_id} [delete]
func (h *TourHandler) DeleteItineraryItem(c echo.Context) error {
	err := h.svc.DeleteItineraryItem(c.Request().Context(), c.Param("package_id"), c.Param("item_id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "itinerary item not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete itinerary item")
	}
	return c.NoContent(http.StatusNoContent)
}
