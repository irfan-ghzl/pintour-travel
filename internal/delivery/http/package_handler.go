package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	pkgsvc "github.com/irfan-ghzl/pintour-travel/internal/application/package"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
)

// ensure strconv is not unused
var _ = strconv.Itoa

type PackageHandler struct{ svc *pkgsvc.Service }

func NewPackageHandler(svc *pkgsvc.Service) *PackageHandler { return &PackageHandler{svc} }

// ListPackages godoc
// @Summary      Daftar paket wisata publik
// @Tags         catalog
// @Produce      json
// @Param        category  query string false "Kategori (reguler/umroh/halal/honeymoon)"
// @Param        destination query string false "Destinasi"
// @Param        price_min query number false "Harga minimum"
// @Param        price_max query number false "Harga maksimum"
// @Param        page query int false "Halaman" default(1)
// @Param        per_page query int false "Per halaman" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /packages [get]
func (h *PackageHandler) ListPackages(c echo.Context) error {
	f := domainPkg.Filter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryInt(c, "per_page", 20),
	}
	active := true
	f.IsActive = &active

	if v := c.QueryParam("category"); v != "" {
		f.Category = &v
	}
	if v := c.QueryParam("destination"); v != "" {
		f.Destination = &v
	}
	if v := c.QueryParam("price_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.PriceMin = &n
		}
	}
	if v := c.QueryParam("price_max"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.PriceMax = &n
		}
	}
	if v := c.QueryParam("departure_month"); v != "" {
		f.DepartureMonth = &v
	}
	// FR-CAT-02: filter durasi perjalanan
	if v := c.QueryParam("duration_min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.DurationMin = &n
		}
	}
	if v := c.QueryParam("duration_max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.DurationMax = &n
		}
	}

	packages, total, err := h.svc.ListPackages(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, pageResponse(packages, total, f.Page, f.PerPage))
}

// GetPackageBySlug godoc
// @Summary      Detail paket berdasarkan slug
// @Tags         catalog
// @Produce      json
// @Param        slug path string true "Slug paket"
// @Success      200 {object} map[string]interface{}
// @Router       /packages/{slug} [get]
func (h *PackageHandler) GetPackageBySlug(c echo.Context) error {
	p, err := h.svc.GetPackageBySlug(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return notFound(c, "paket tidak ditemukan")
	}
	batches, _ := h.svc.ListBatches(c.Request().Context(), p.ID)
	images, _ := h.svc.ListImages(c.Request().Context(), p.ID)
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"package": p, "batches": batches, "images": images,
	}))
}

// ── Admin handlers ────────────────────────────────────────────────────────────

// AdminListPackages godoc
// @Summary      Daftar paket (admin)
// @Tags         admin-packages
// @Security     BearerAuth
// @Produce      json
// @Param        category query string false "Kategori"
// @Param        page query int false "Halaman" default(1)
// @Success      200 {object} map[string]interface{}
// @Router       /admin/packages [get]
func (h *PackageHandler) AdminListPackages(c echo.Context) error {
	f := domainPkg.Filter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryInt(c, "per_page", 20),
	}
	if v := c.QueryParam("category"); v != "" {
		f.Category = &v
	}
	packages, total, err := h.svc.ListPackages(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, pageResponse(packages, total, f.Page, f.PerPage))
}

// CreatePackage godoc
// @Summary      Buat paket baru
// @Tags         admin-packages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body domainPkg.Package true "Data paket"
// @Success      201 {object} map[string]interface{}
// @Router       /admin/packages [post]
func (h *PackageHandler) CreatePackage(c echo.Context) error {
	var p domainPkg.Package
	if err := bindJSON(c, &p); err != nil {
		return badRequest(c, err.Error())
	}
	userID := claimUserID(c)
	p.CreatedBy = userID
	if err := h.svc.CreatePackage(c.Request().Context(), &p); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(p))
}

// UpdatePackage godoc
// @Summary      Update paket
// @Tags         admin-packages
// @Security     BearerAuth
// @Param        id path string true "Package ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/packages/{id} [put]
func (h *PackageHandler) UpdatePackage(c echo.Context) error {
	existing, err := h.svc.GetPackage(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "paket tidak ditemukan")
	}
	if err := bindJSON(c, existing); err != nil {
		return badRequest(c, err.Error())
	}
	existing.ID = c.Param("id")
	if err := h.svc.UpdatePackage(c.Request().Context(), existing); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(existing))
}

// DeletePackage godoc
// @Summary      Hapus paket (soft delete)
// @Tags         admin-packages
// @Security     BearerAuth
// @Param        id path string true "Package ID"
// @Success      204
// @Router       /admin/packages/{id} [delete]
func (h *PackageHandler) DeletePackage(c echo.Context) error {
	if err := h.svc.DeletePackage(c.Request().Context(), c.Param("id")); err != nil {
		return serverErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Batch handlers ────────────────────────────────────────────────────────────

func (h *PackageHandler) ListBatches(c echo.Context) error {
	batches, err := h.svc.ListBatches(c.Request().Context(), c.Param("package_id"))
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(batches))
}

func (h *PackageHandler) CreateBatch(c echo.Context) error {
	var b domainPkg.PackageBatch
	if err := bindJSON(c, &b); err != nil {
		return badRequest(c, err.Error())
	}
	b.PackageID = c.Param("package_id")
	if err := h.svc.CreateBatch(c.Request().Context(), &b); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(b))
}

func (h *PackageHandler) UpdateBatch(c echo.Context) error {
	b, err := h.svc.GetBatch(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "batch tidak ditemukan")
	}
	if err := bindJSON(c, b); err != nil {
		return badRequest(c, err.Error())
	}
	b.ID = c.Param("id")
	if err := h.svc.UpdateBatch(c.Request().Context(), b); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(b))
}

// ── Image handlers ────────────────────────────────────────────────────────────

func (h *PackageHandler) ListImages(c echo.Context) error {
	images, err := h.svc.ListImages(c.Request().Context(), c.Param("package_id"))
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(images))
}

func (h *PackageHandler) AddImage(c echo.Context) error {
	var img domainPkg.PackageImage
	if err := bindJSON(c, &img); err != nil {
		return badRequest(c, err.Error())
	}
	img.PackageID = c.Param("package_id")
	if err := h.svc.AddImage(c.Request().Context(), &img); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(img))
}

func (h *PackageHandler) DeleteImage(c echo.Context) error {
	if err := h.svc.DeleteImage(c.Request().Context(), c.Param("image_id")); err != nil {
		return serverErr(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func bindJSON(c echo.Context, v interface{}) error {
	return json.NewDecoder(c.Request().Body).Decode(v)
}
