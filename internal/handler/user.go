package handler

import (
	"net/http"

	"github.com/irfan-ghzl/pintour-travel/internal/service"
	"github.com/labstack/echo/v4"
)

// UserHandler handles authentication and user management requests.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Login godoc
//
//	@Summary     Authenticate a user
//	@Description Validates email/password and returns a JWT
//	@Tags        auth
//	@Accept      json
//	@Produce     json
//	@Param       body body service.LoginRequest true "Login credentials"
//	@Success     200 {object} service.LoginResponse
//	@Failure     401 {object} map[string]interface{}
//	@Router      /api/v1/auth/login [post]
func (h *UserHandler) Login(c echo.Context) error {
	var req service.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	resp, statusCode, err := h.svc.Login(c.Request().Context(), nil, req)
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
//	@Router      /api/v1/auth/me [get]
func (h *UserHandler) Me(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	role, _ := c.Get("user_role").(string)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"role":    role,
	})
}
