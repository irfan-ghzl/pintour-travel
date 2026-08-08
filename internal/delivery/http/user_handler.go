package httpdelivery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// secureToken returns a cryptographically-random 256-bit token as hex.
func secureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

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
	if err := bindJSON(c, &req); err != nil {
		return invalidPayload(c, err, "email dan password harus diisi")
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

// resetTokenTTL is how long a reset link stays redeemable.
const resetTokenTTL = time.Hour

// resetTokens holds the password-reset tokens for the process: a link issued by
// one request is redeemed by another, so the store outlives both. Production
// deployments would keep these in Redis or the database; in memory they are
// only safe because the sweeper below evicts them.
var resetTokens = newResetTokenStore()

type resetEntry struct {
	email  string
	expiry time.Time
}

// resetTokenStore is an in-memory map of reset token → account, kept from
// growing without bound by a periodic sweep — the same shape the rate limiter
// uses to evict its per-IP buckets. Most tokens are never redeemed (a user asks
// for a link, then remembers the password), so without the sweep every
// forgot-password request would cost the process memory permanently.
type resetTokenStore struct {
	mu     sync.Mutex
	tokens map[string]resetEntry

	sweeperOnce sync.Once
}

func newResetTokenStore() *resetTokenStore {
	return &resetTokenStore{tokens: map[string]resetEntry{}}
}

// issue records a token that unlocks email until expiry.
func (s *resetTokenStore) issue(token, email string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = resetEntry{email: email, expiry: expiry}
}

// consume returns the entry behind token and removes it, so a reset link works
// exactly once. An expired entry is removed too, and reported as absent.
func (s *resetTokenStore) consume(token string, now time.Time) (resetEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, found := s.tokens[token]
	if !found {
		return resetEntry{}, false
	}
	delete(s.tokens, token)
	if now.After(entry.expiry) {
		return resetEntry{}, false
	}
	return entry, true
}

// sweep drops every entry that expired before now and reports how many went.
func (s *resetTokenStore) sweep(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	dropped := 0
	for token, entry := range s.tokens {
		if now.After(entry.expiry) {
			delete(s.tokens, token)
			dropped++
		}
	}
	return dropped
}

func (s *resetTokenStore) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tokens)
}

// startSweeper launches the eviction loop, at most once per store however many
// times it is called — RegisterRoutes runs once per process in a deployment but
// once per case in the tests, and each extra sweeper would be a leaked
// goroutine.
func (s *resetTokenStore) startSweeper(every time.Duration) {
	s.sweeperOnce.Do(func() {
		safe.Every("penyapu token reset password", every, func() {
			s.sweep(time.Now())
		})
	})
}

// ForgotPassword godoc
// @Summary      Kirim link reset password via email Resend (FR-USER-04)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /auth/forgot-password [post]
func (h *UserHandler) ForgotPassword(c echo.Context) error {
	var body struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "email harus diisi")
	}
	u, err := h.repo.GetByEmail(c.Request().Context(), body.Email)
	if err != nil || u == nil {
		// Don't reveal whether the email exists
		return c.JSON(http.StatusOK, ok(map[string]string{
			"message": "Jika email terdaftar, link reset password akan dikirimkan",
		}))
	}

	token, err := secureToken()
	if err != nil {
		return serverErr(c, err)
	}
	resetTokens.issue(token, u.Email, time.Now().Add(resetTokenTTL))

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", h.appURL, token)
	safe.Go("kirim email reset password", func() {
		_ = h.email.SendResetPassword(context.Background(), u.Email, u.Name, resetLink)
	})

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
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "token dan password baru harus diisi")
	}

	entry, found := resetTokens.consume(body.Token, time.Now())
	if !found {
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
		Name     string `json:"name" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
		Role     string `json:"role" validate:"required,staff_role"`
		Phone    string `json:"phone" validate:"omitempty,phone_id"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "nama, email, password, dan role harus diisi")
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
		Email string `json:"email" validate:"omitempty,email"`
		Role  string `json:"role" validate:"omitempty,staff_role"`
		Phone string `json:"phone" validate:"omitempty,phone_id"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "format tidak valid")
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
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "password baru harus diisi")
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
