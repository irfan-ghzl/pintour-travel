package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// UploadHandler exposes file upload endpoints backed by Supabase Storage.
type UploadHandler struct {
	storage *service.StorageService
}

func NewUploadHandler(storage *service.StorageService) *UploadHandler {
	return &UploadHandler{storage: storage}
}

// UploadDocument godoc
// @Summary      Upload dokumen peserta (multipart) ke Supabase private bucket (§16.2)
// @Tags         upload
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Param        file formData file true "File dokumen (max 5MB)"
// @Success      201 {object} map[string]interface{}
// @Router       /portal/upload/document [post]
func (h *UploadHandler) UploadDocument(c echo.Context) error {
	if !h.storage.Enabled() {
		return c.JSON(http.StatusServiceUnavailable,
			errResponse("STORAGE_UNAVAILABLE", "Storage tidak terkonfigurasi (gunakan URL manual sementara)"))
	}
	pid := portalParticipantID(c)
	fh, err := c.FormFile("file")
	if err != nil {
		return badRequest(c, "file harus diisi")
	}
	res, err := h.storage.Upload(c.Request().Context(), "participant-documents", pid, fh)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("UPLOAD_FAILED", err.Error()))
	}
	return c.JSON(http.StatusCreated, ok(map[string]string{
		"path":      res.Path,
		"file_name": fh.Filename,
	}))
}

// SignedURL godoc
// @Summary      Dapatkan signed URL untuk akses file private (1 jam, §19.2)
// @Tags         upload
// @Security     BearerAuth
// @Param        bucket query string true "Nama bucket"
// @Param        path query string true "Path file"
// @Success      200 {object} map[string]interface{}
// @Router       /portal/signed-url [get]
func (h *UploadHandler) SignedURL(c echo.Context) error {
	if !h.storage.Enabled() {
		return c.JSON(http.StatusServiceUnavailable,
			errResponse("STORAGE_UNAVAILABLE", "Storage tidak terkonfigurasi"))
	}
	bucket := c.QueryParam("bucket")
	path := c.QueryParam("path")
	if bucket == "" || path == "" {
		return badRequest(c, "bucket dan path harus diisi")
	}
	url, err := h.storage.SignedURL(c.Request().Context(), bucket, path, 3600) // 1 jam per §19.2
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errResponse("SIGN_FAILED", err.Error()))
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"url": url}))
}

// UploadPackageImage godoc
// @Summary      Upload foto paket ke Supabase public bucket (admin) (§16.2)
// @Tags         upload
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Param        package_id path string true "Package ID"
// @Param        file formData file true "File gambar"
// @Success      201 {object} map[string]interface{}
// @Router       /admin/packages/{package_id}/images/upload [post]
func (h *UploadHandler) UploadPackageImage(c echo.Context) error {
	if !h.storage.Enabled() {
		return c.JSON(http.StatusServiceUnavailable,
			errResponse("STORAGE_UNAVAILABLE", "Storage tidak terkonfigurasi"))
	}
	packageID := c.Param("package_id")
	fh, err := c.FormFile("file")
	if err != nil {
		return badRequest(c, "file harus diisi")
	}
	res, err := h.storage.Upload(c.Request().Context(), "package-images", packageID, fh)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("UPLOAD_FAILED", err.Error()))
	}
	return c.JSON(http.StatusCreated, ok(map[string]string{
		"path":       res.Path,
		"public_url": res.PublicURL,
	}))
}

// UploadPaymentProof godoc
// @Summary      Upload bukti transfer peserta (multipart) (AC-INV-02)
// @Tags         upload
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Param        file formData file true "Bukti transfer"
// @Success      201 {object} map[string]interface{}
// @Router       /portal/upload/payment-proof [post]
func (h *UploadHandler) UploadPaymentProof(c echo.Context) error {
	if !h.storage.Enabled() {
		return c.JSON(http.StatusServiceUnavailable,
			errResponse("STORAGE_UNAVAILABLE", "Storage tidak terkonfigurasi"))
	}
	pid := portalParticipantID(c)
	fh, err := c.FormFile("file")
	if err != nil {
		return badRequest(c, "file harus diisi")
	}
	res, err := h.storage.Upload(c.Request().Context(), "payment-proofs", pid, fh)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("UPLOAD_FAILED", err.Error()))
	}
	return c.JSON(http.StatusCreated, ok(map[string]string{"path": res.Path}))
}
