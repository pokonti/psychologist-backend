package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"gorm.io/gorm"
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

// AdminGetUserDetails godoc
// @Summary      Admin: View specific user profile
// @Description  Allows admin to see the full profile details of any user, including sensitive data.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID (UUID)"
// @Success      200  {object}  models.UserProfile
// @Failure      403  {object}  models.ErrorResponse "Admin access required"
// @Failure      404  {object}  models.ErrorResponse "User not found"
// @Router       /admin/users/{id} [get]
func (h *ProfileHandler) AdminGetUserDetails(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	targetID := c.Param("id")

	profile, err := h.Repo.GetByID(c.Request.Context(), targetID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
