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
