package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/internal/models"
	"github.com/pokonti/psychologist-backend/auth-service/internal/utils"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

type AdminAddUserInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=student psychologist admin"`
	FullName string `json:"full_name" binding:"required"`
}

type BlockUserInput struct {
	Blocked bool `json:"blocked"`
}

// AdminAddUser godoc
// @Summary      Admin: Add a new user
// @Description  Admin directly creates a verified user (skips email verification).
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body AdminAddUserInput true "User Info"
// @Success      200 {object} models.MessageResponse
// @Failure      401 {object} models.ErrorResponse
// @Router       /admin/users [post]
func (ac *AuthController) AdminAddUser(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	var input AdminAddUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	hashedPassword, _ := utils.HashPassword(input.Password)
	user := models.User{
		ID:         uuid.NewString(),
		Email:      input.Email,
		Password:   hashedPassword,
		Role:       input.Role,
		IsVerified: true,
		IsBlocked:  false,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Email already exists"})
		return
	}

	_, err := ac.UserClient.CreateUserProfile(c.Request.Context(), &userprofile.CreateUserProfileRequest{
		Id:       user.ID,
		Email:    user.Email,
		Role:     user.Role,
		FullName: input.FullName,
	})

	if err != nil {
		config.DB.Unscoped().Delete(&user)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create user profile"})
		return
	}

	c.JSON(http.StatusCreated, models.MessageResponse{Message: fmt.Sprintf("User created successfully with ID: %s", user.ID)})
}

// AdminBlockUser godoc
// @Summary      Admin: Block or Unblock a user
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        request body BlockUserInput true "Block Status"
// @Success      200 {object} map[string]string
// @Router       /admin/users/{id}/block [patch]
func (ac *AuthController) AdminBlockUser(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	userID := c.Param("id")
	var input BlockUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	res := config.DB.Model(&models.User{}).Where("id = ?", userID).Update("is_blocked", input.Blocked)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}

	status := "unblocked"
	if input.Blocked {
		status = "blocked"
	}
	c.JSON(http.StatusOK, models.MessageResponse{Message: fmt.Sprintf("User successfully %s", status)})
}
