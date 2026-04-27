package handler

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GalleryHandler handles HTTP requests for package galleries.
type GalleryHandler struct {
	db *sql.DB
}

// NewGalleryHandler creates a new GalleryHandler.
func NewGalleryHandler(db *sql.DB) *GalleryHandler {
	return &GalleryHandler{db: db}
}

// ListGallery godoc
//
//	@Summary     List gallery images for a package
//	@Tags        gallery
//	@Produce     json
//	@Param       package_id path string true "Tour Package ID"
//	@Success     200 {array} map[string]interface{}
//	@Router      /api/v1/packages/{package_id}/gallery [get]
func (h *GalleryHandler) ListGallery(c echo.Context) error {
	packageID := c.Param("package_id")

	rows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT id, image_url, caption, sort_order
		FROM package_galleries
		WHERE tour_package_id = $1
		ORDER BY sort_order ASC, created_at ASC`, packageID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch gallery")
	}
	defer rows.Close()

	var gallery []map[string]interface{}
	for rows.Next() {
		var id, imageURL string
		var caption *string
		var sortOrder int
		if err := rows.Scan(&id, &imageURL, &caption, &sortOrder); err != nil {
			continue
		}
		gallery = append(gallery, map[string]interface{}{
			"id": id, "image_url": imageURL, "caption": caption, "sort_order": sortOrder,
		})
	}
	if gallery == nil {
		gallery = []map[string]interface{}{}
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
func (h *GalleryHandler) AddGalleryImage(c echo.Context) error {
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

	var id string
	err := h.db.QueryRowContext(c.Request().Context(), `
		INSERT INTO package_galleries (tour_package_id, image_url, caption, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		packageID, body.ImageURL, caption, body.SortOrder,
	).Scan(&id)
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
//	@Produce     json
//	@Security    BearerAuth
//	@Param       package_id path string true "Tour Package ID"
//	@Param       image_id   path string true "Image ID"
//	@Success     204
//	@Router      /api/v1/admin/packages/{package_id}/gallery/{image_id} [delete]
func (h *GalleryHandler) DeleteGalleryImage(c echo.Context) error {
	imageID := c.Param("image_id")
	packageID := c.Param("package_id")

	res, err := h.db.ExecContext(c.Request().Context(),
		`DELETE FROM package_galleries WHERE id=$1 AND tour_package_id=$2`, imageID, packageID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete image")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "image not found")
	}
	return c.NoContent(http.StatusNoContent)
}
