package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v4"

	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

type TourLeaderHandler struct {
	repo domainUser.TourLeaderRepository
}

func NewTourLeaderHandler(repo domainUser.TourLeaderRepository) *TourLeaderHandler {
	return &TourLeaderHandler{repo}
}

// List godoc
// @Summary      Daftar profil tour leader (FR-BRIEF-03)
// @Tags         tour-leaders
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /admin/tour-leaders [get]
func (h *TourLeaderHandler) List(c echo.Context) error {
	list, err := h.repo.List(c.Request().Context())
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(list))
}

// GetByUser godoc
// @Summary      Get profil tour leader by user_id
// @Tags         tour-leaders
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/tour-leaders/{user_id} [get]
func (h *TourLeaderHandler) GetByUser(c echo.Context) error {
	tl, err := h.repo.GetByUserID(c.Request().Context(), c.Param("user_id"))
	if err != nil {
		return notFound(c, "profil tour leader tidak ditemukan")
	}
	return c.JSON(http.StatusOK, ok(tl))
}

// Upsert godoc
// @Summary      Create/Update profil tour leader (admin)
// @Tags         tour-leaders
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Accept       json
// @Success      200 {object} map[string]interface{}
// @Router       /admin/tour-leaders/{user_id} [put]
func (h *TourLeaderHandler) Upsert(c echo.Context) error {
	var tl domainUser.TourLeader
	if err := bindJSON(c, &tl); err != nil {
		return invalidPayload(c, err, "format tidak valid")
	}
	tl.UserID = c.Param("user_id")

	// Try update first, create if not exists
	existing, err := h.repo.GetByUserID(c.Request().Context(), tl.UserID)
	if err != nil {
		// Create new
		if e := h.repo.Create(c.Request().Context(), &tl); e != nil {
			return serverErr(c, e)
		}
		return c.JSON(http.StatusCreated, ok(tl))
	}
	existing.Bio = tl.Bio
	existing.PhotoPath = tl.PhotoPath
	existing.ExperienceYears = tl.ExperienceYears
	existing.Specialization = tl.Specialization
	existing.EmergencyPhone = tl.EmergencyPhone
	if err := h.repo.Update(c.Request().Context(), existing); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(existing))
}
