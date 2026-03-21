package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
)

// ListAllUsers godoc
// @Summary      Admin: List all users
// @Description  Returns all profiles (Students, Psychologists, Admins).
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.UserProfile
// @Failure      401 {object} models.ErrorResponse
// @Router       /admin/users [get]
func (h *ProfileHandler) ListAllUsers(c *gin.Context) {
	role := c.GetHeader("X-User-Role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	var users []models.UserProfile
	if err := config.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetAllPsychologists godoc
// @Summary      List all psychologists
// @Description  Returns a list of all users whose role is 'psychologist'.
// @Tags         psychologists
// @Produce      json
// @Success      200  {array}   models.UserProfile
// @Failure      500  {object}  models.ErrorResponse "database error"
// @Security     BearerAuth
// @Router       /users/psychologists [get]
func (h *ProfileHandler) GetAllPsychologists(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	profiles, err := h.Repo.GetAllPsychologists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Database error:" + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, profiles)
}
