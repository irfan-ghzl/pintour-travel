package httpdelivery

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	svc    *usersvc.UserService
	repo   domainUser.Repository
	email  *service.EmailService
	appURL string
	prod   bool // true di production → cookie Secure
	expiry int  // JWT expiry hours
}

func NewUserHandler(svc *usersvc.UserService, repo domainUser.Repository, email *service.EmailService, appURL string, prod bool, expiryHours int) *UserHandler {
	return &UserHandler{svc: svc, repo: repo, email: email, appURL: appURL, prod: prod, expiry: expiryHours}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func (h *UserHandler) Login(c echo.Context) error {
	var req usersvc.LoginRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "format tidak valid")
	}
	if req.Email == "" || req.Password == "" {
		return badRequest(c, "email dan password harus diisi")
	}
	resp, statusCode, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return c.JSON(statusCode, errResponse("UNAUTHORIZED", err.Error()))
	}

	// §19.1: simpan JWT di httpOnly cookie untuk mencegah XSS
	expirySec := h.expiry * 3600
	if expirySec <= 0 {
		expirySec = 8 * 3600
	}
	SetSessionCookie(c, resp.Token, expirySec, h.prod)

	return c.JSON(http.StatusOK, ok(resp))
}

// Logout godoc
// @Summary      Logout admin/staff — clear httpOnly session cookie (§19.1)
// @Tags         auth
// @Success      200 {object} map[string]interface{}
// @Router       /auth/logout [post]
func (h *UserHandler) Logout(c echo.Context) error {
	ClearSessionCookie(c)
	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Logout berhasil"}))
}

func (h *UserHandler) Me(c echo.Context) error {
	userID := claimUserID(c)
	role := claimRole(c)
	u, err := h.repo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusOK, ok(map[string]string{"user_id": userID, "role": role}))
	}
	return c.JSON(http.StatusOK, ok(map[string]interface{}{
		"user_id": u.ID,
		"name":    u.Name,
		"email":   u.Email,
		"role":    u.Role,
		"phone":   u.Phone,
	}))
}

// ── Reset Password (FR-USER-04) ───────────────────────────────────────────────

// Simple in-memory reset token store (TTL 1 jam). Production: use Redis/DB.
var (
	resetTokenMu    sync.Mutex
	resetTokenStore = map[string]resetEntry{} // token → {email, expiry}
)

type resetEntry struct {
	email  string
	expiry time.Time
}

// ForgotPassword godoc
// @Summary      Kirim link reset password via email Resend (FR-USER-04)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /auth/forgot-password [post]
func (h *UserHandler) ForgotPassword(c echo.Context) error {
	var body struct{ Email string `json:"email"` }
	if err := bindJSON(c, &body); err != nil || body.Email == "" {
		return badRequest(c, "email harus diisi")
	}
	u, err := h.repo.GetByEmail(c.Request().Context(), body.Email)
	if err != nil || u == nil {
		// Don't reveal whether the email exists
		return c.JSON(http.StatusOK, ok(map[string]string{
			"message": "Jika email terdaftar, link reset password akan dikirimkan",
		}))
	}

	token := fmt.Sprintf("%x", time.Now().UnixNano())
	resetTokenMu.Lock()
	resetTokenStore[token] = resetEntry{email: u.Email, expiry: time.Now().Add(time.Hour)}
	resetTokenMu.Unlock()

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", h.appURL, token)
	go func() {
		_ = h.email.SendResetPassword(context.Background(), u.Email, u.Name, resetLink)
	}()

	return c.JSON(http.StatusOK, ok(map[string]string{
		"message": "Jika email terdaftar, link reset password akan dikirimkan",
	}))
}

// ResetPassword godoc
// @Summary      Ubah password baru dengan token reset (FR-USER-04)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /auth/reset-password [post]
func (h *UserHandler) ResetPassword(c echo.Context) error {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := bindJSON(c, &body); err != nil || body.Token == "" || body.Password == "" {
		return badRequest(c, "token dan password baru harus diisi")
	}
	if len(body.Password) < 8 {
		return badRequest(c, "password minimal 8 karakter")
	}

	resetTokenMu.Lock()
	entry, found := resetTokenStore[body.Token]
	if found {
		delete(resetTokenStore, body.Token)
	}
	resetTokenMu.Unlock()

	if !found || time.Now().After(entry.expiry) {
		return c.JSON(http.StatusUnprocessableEntity,
			errResponse("TOKEN_EXPIRED", "Token tidak valid atau sudah kadaluarsa"))
	}

	u, err := h.repo.GetByEmail(c.Request().Context(), entry.email)
	if err != nil || u == nil {
		return notFound(c, "pengguna tidak ditemukan")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return serverErr(c, err)
	}
	u.Password = string(hashed)
	if err := h.repo.Update(c.Request().Context(), u); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Password berhasil diubah, silakan login kembali"}))
}

// ── User Management (FR-USER-03) ──────────────────────────────────────────────

// ListUsers godoc
// @Summary      Daftar pengguna sistem (FR-USER-03)
// @Tags         users
// @Security     BearerAuth
// @Param        role query string false "Filter role (super_admin/admin/konsultan/tour_leader)"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/users [get]
func (h *UserHandler) ListUsers(c echo.Context) error {
	role := c.QueryParam("role")
	var users []domainUser.User
	var err error
	if role != "" {
		users, err = h.repo.ListByRole(c.Request().Context(), role)
		if err != nil {
			return serverErr(c, err)
		}
	} else {
		for _, r := range []string{"super_admin", "admin", "konsultan", "tour_leader"} {
			us, e := h.repo.ListByRole(c.Request().Context(), r)
			if e == nil {
				users = append(users, us...)
			}
		}
	}
	safe := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		safe = append(safe, userToMap(u))
	}
	return c.JSON(http.StatusOK, ok(safe))
}

// CreateUser godoc
// @Summary      Buat akun pengguna (super_admin only) (FR-USER-03)
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Success      201 {object} map[string]interface{}
// @Router       /admin/users [post]
func (h *UserHandler) CreateUser(c echo.Context) error {
	if claimRole(c) != "super_admin" {
		return forbidden(c)
	}
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Phone    string `json:"phone"`
	}
	if err := bindJSON(c, &body); err != nil {
		return badRequest(c, "format tidak valid")
	}
	if body.Name == "" || body.Email == "" || body.Password == "" || body.Role == "" {
		return badRequest(c, "nama, email, password, dan role harus diisi")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return serverErr(c, err)
	}
	u := &domainUser.User{
		Name: body.Name, Email: body.Email, Password: string(hashed),
		Role: body.Role, Phone: body.Phone, IsActive: true,
	}
	if err := h.repo.Create(c.Request().Context(), u); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(userToMap(*u)))
}

// UpdateUser godoc
// @Summary      Edit data pengguna (super_admin only)
// @Tags         users
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/users/{id} [put]
func (h *UserHandler) UpdateUser(c echo.Context) error {
	if claimRole(c) != "super_admin" {
		return forbidden(c)
	}
	existing, err := h.repo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "pengguna tidak ditemukan")
	}
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
		Phone string `json:"phone"`
	}
	if err := bindJSON(c, &body); err != nil {
		return badRequest(c, "format tidak valid")
	}
	if body.Name != "" {
		existing.Name = body.Name
	}
	if body.Email != "" {
		existing.Email = body.Email
	}
	if body.Role != "" {
		existing.Role = body.Role
	}
	if body.Phone != "" {
		existing.Phone = body.Phone
	}
	if err := h.repo.Update(c.Request().Context(), existing); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(userToMap(*existing)))
}

// DeactivateUser godoc
// @Summary      Nonaktifkan akun (soft delete, super_admin only)
// @Tags         users
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/users/{id}/deactivate [patch]
func (h *UserHandler) DeactivateUser(c echo.Context) error {
	if claimRole(c) != "super_admin" {
		return forbidden(c)
	}
	if err := h.repo.Deactivate(c.Request().Context(), c.Param("id")); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Akun berhasil dinonaktifkan"}))
}

// ResetPasswordAdmin godoc
// @Summary      Reset password user oleh super admin
// @Tags         users
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/users/{id}/reset-password [patch]
func (h *UserHandler) ResetPasswordAdmin(c echo.Context) error {
	if claimRole(c) != "super_admin" {
		return forbidden(c)
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := bindJSON(c, &body); err != nil || body.Password == "" {
		return badRequest(c, "password baru harus diisi")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return serverErr(c, err)
	}
	u, err := h.repo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return notFound(c, "pengguna tidak ditemukan")
	}
	u.Password = string(hashed)
	if err := h.repo.Update(c.Request().Context(), u); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(map[string]string{"message": "Password berhasil direset"}))
}

func userToMap(u domainUser.User) map[string]interface{} {
	return map[string]interface{}{
		"id": u.ID, "name": u.Name, "email": u.Email,
		"role": u.Role, "phone": u.Phone, "is_active": u.IsActive,
		"created_at": u.CreatedAt,
	}
}
