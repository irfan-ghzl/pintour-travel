package httpdelivery

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	participantsvc "github.com/irfan-ghzl/pintour-travel/internal/application/participant"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// UploadHandler exposes file upload endpoints backed by Supabase Storage, plus
// the signed-URL endpoint that hands out read access to what was uploaded.
type UploadHandler struct {
	storage *service.StorageService
	files   privateFileResolver
	// participants resolves which tours a portal identity owns, which is what
	// "their own file" means for a returning customer whose account spans
	// several trips (v3.0).
	participants *participantsvc.Service
}

func NewUploadHandler(
	storage *service.StorageService,
	documents document.Repository,
	invoices *invoicesvc.Service,
	participants *participantsvc.Service,
) *UploadHandler {
	return &UploadHandler{
		storage:      storage,
		files:        privateFileResolver{documents: documents, invoices: invoices},
		participants: participants,
	}
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
	res, err := h.storage.Upload(c.Request().Context(), bucketParticipantDocuments, pid, fh)
	if err != nil {
		return uploadErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(map[string]string{
		"path":      res.Path,
		"file_name": fh.Filename,
	}))
}

// A file belonging to another participant answers 404, not 403: a 403 would
// confirm that the file exists, which is most of what an attacker enumerating
// identifiers is after.
//
// SignedURLForPortal godoc
// @Summary      Signed URL berkas privat milik peserta sendiri (1 jam, §19.2)
// @Tags         upload
// @Security     BearerAuth
// @Param        type query string true "Jenis berkas: document | payment_proof"
// @Param        id query string true "ID sumber daya (dokumen / bukti bayar)"
// @Success      200 {object} map[string]interface{}
// @Router       /portal/signed-url [get]
func (h *UploadHandler) SignedURLForPortal(c echo.Context) error {
	file, err := h.resolveRequestedFile(c)
	if err != nil {
		return privateFileError(c, err)
	}
	owned, err := portalOwnsParticipant(c, h.participants, file.ParticipantID)
	if err != nil {
		return serverErr(c, err)
	}
	if !owned {
		return notFound(c, "berkas tidak ditemukan")
	}
	return h.signPrivateFile(c, file)
}

// Reaching this handler is the whole authorization. The route sits on the ops
// group, so the roles that get here are exactly the ones §5.3 lets list and
// review participant documents and payment proofs — reviewing them across
// participants is the job. Roles outside that group are stopped by RequireRole
// before any resource is resolved.
//
// SignedURLForStaff godoc
// @Summary      Signed URL berkas privat peserta untuk staf (1 jam, §19.2)
// @Tags         upload
// @Security     BearerAuth
// @Param        type query string true "Jenis berkas: document | payment_proof"
// @Param        id query string true "ID sumber daya (dokumen / bukti bayar)"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/signed-url [get]
func (h *UploadHandler) SignedURLForStaff(c echo.Context) error {
	file, err := h.resolveRequestedFile(c)
	if err != nil {
		return privateFileError(c, err)
	}
	return h.signPrivateFile(c, file)
}

// uploadErr answers a failed upload.
//
// A storage backend that cannot be reached is not the uploader's mistake, so it
// does not get 400: it answers 503, which is both truthful and the status the
// browser fallback watches for to offer a manual URL. The underlying error goes
// to the log rather than to the response — it carries the bucket host and the
// full object path, which is our infrastructure, not the participant's business.
// It used to travel the other way: printed in the upload dialog, recorded
// nowhere.
func uploadErr(c echo.Context, err error) error {
	if errors.Is(err, service.ErrStorageUnreachable) {
		c.Logger().Error(err)
		return c.JSON(http.StatusServiceUnavailable,
			errResponse("STORAGE_UNAVAILABLE", "Storage tidak terkonfigurasi (gunakan URL manual sementara)"))
	}
	return c.JSON(http.StatusBadRequest, errResponse("UPLOAD_FAILED", err.Error()))
}

// resolveRequestedFile reads the resource the caller named and resolves where it
// is stored. What comes back is what the access check runs on; the request
// itself contributes nothing but the identifier.
func (h *UploadHandler) resolveRequestedFile(c echo.Context) (privateFile, error) {
	kind, id := c.QueryParam("type"), c.QueryParam("id")
	if kind == "" || id == "" {
		return privateFile{}, errMissingFileRef
	}
	return h.files.resolve(c.Request().Context(), kind, id)
}

// signPrivateFile mints the URL, and runs only once the caller has been allowed
// the file. The storage-configured check is here rather than at the top of the
// handlers deliberately: a caller who may not read the file gets the same 404
// whether or not this deployment has storage keys.
func (h *UploadHandler) signPrivateFile(c echo.Context, file privateFile) error {
	// A path recorded as an absolute URL is already reachable and has no bucket
	// object behind it — that is the manual-URL fallback a deployment without
	// storage keys writes, and what the seed data contains. Signing it produced a
	// request for ".../object/sign/<bucket>/https://example.com/..." and a 500 on
	// the one path the caller was entitled to. The client already understood this
	// shape; the server, which took over the decision, did not.
	if isAbsoluteURL(file.Path) {
		return c.JSON(http.StatusOK, ok(map[string]string{"url": file.Path}))
	}
	if !h.storage.Enabled() {
		return c.JSON(http.StatusServiceUnavailable,
			errResponse("STORAGE_UNAVAILABLE", "Storage tidak terkonfigurasi"))
	}
	url, err := h.storage.SignedURL(c.Request().Context(), file.Bucket, file.Path, signedURLTTL)
	if err != nil {
		// The bucket being unreachable is an upstream outage, not a fault in this
		// server: 503 says so, and says it without the 500's implication that the
		// request itself broke something.
		if errors.Is(err, service.ErrStorageUnreachable) {
			c.Logger().Error(err)
			return c.JSON(http.StatusServiceUnavailable,
				errResponse("STORAGE_UNAVAILABLE", "Storage sedang tidak dapat diakses"))
		}
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"url": url}))
}

// privateFileError maps a resolution failure onto a response. A missing row and
// a file belonging to someone else deliberately produce the same 404 body.
func privateFileError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, errMissingFileRef):
		return badRequest(c, "type dan id berkas harus diisi")
	case errors.Is(err, errUnknownFileKind):
		return badRequest(c, "jenis berkas harus document atau payment_proof")
	case errors.Is(err, sql.ErrNoRows):
		return notFound(c, "berkas tidak ditemukan")
	}
	return serverErr(c, err)
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
		return uploadErr(c, err)
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
	res, err := h.storage.Upload(c.Request().Context(), bucketPaymentProofs, pid, fh)
	if err != nil {
		return uploadErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(map[string]string{"path": res.Path}))
}
