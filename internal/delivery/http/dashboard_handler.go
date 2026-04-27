package httpdelivery

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// DashboardHandler handles admin dashboard requests.
type DashboardHandler struct{}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// GetStats godoc
//
//	@Summary     Dashboard statistics (admin)
//	@Tags        dashboard
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/dashboard/stats [get]
func (h *DashboardHandler) GetStats(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"generated_at": time.Now().Format(time.RFC3339),
		"note":         "Connect to a real database to get live stats",
	})
}
