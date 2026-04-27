package httpdelivery

import (
	"net/http"

	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/labstack/echo/v4"
)

// UserHandler handles authentication HTTP requests.
type UserHandler struct {
	svc *usersvc.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *usersvc.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Login godoc
//
//	@Summary     Authenticate a user
//	@Tags        auth
//	@Accept      json
//	@Produce     json
//	@Param       body body usersvc.LoginRequest true "Login credentials"
//	@Success     200 {object} usersvc.LoginResponse
//	@Failure     401 {object} map[string]interface{}
//	@Router      /api/v1/auth/login [post]
func (h *UserHandler) Login(c echo.Context) error {
	var req usersvc.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	resp, statusCode, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(statusCode, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// Me godoc
//
//	@Summary     Get current user info
//	@Tags        auth
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/auth/me [get]
func (h *UserHandler) Me(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"role":    role,
	})
}
