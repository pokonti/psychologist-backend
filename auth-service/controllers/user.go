package controllers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/auth-service/clients"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/models"
	"github.com/pokonti/psychologist-backend/auth-service/utils"
	"github.com/pokonti/psychologist-backend/proto/userprofile"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/middleware"
)

type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required"` // "student", "psychologist"
	FullName string `json:"full_name" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var input RegisterInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate UUID for user
	userID := uuid.NewString()

	user := models.User{
		ID:       userID,
		Email:    input.Email,
		Password: hashedPassword,
		Role:     input.Role,
	}

	// Save in auth DB
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Call user-service via gRPC to create profile
	_, err = clients.UserProfileClient.CreateUserProfile(
		c.Request.Context(),
		&userprofile.CreateUserProfileRequest{
			Id:       user.ID,
			Email:    user.Email,
			Role:     user.Role,
			FullName: input.FullName,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user profile"})
		return
	}

	// Generate JWT with userID + email + role
	token, err := middleware.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
		"token": token,
	})
}

func Login(c *gin.Context) {
	var input LoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := middleware.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
