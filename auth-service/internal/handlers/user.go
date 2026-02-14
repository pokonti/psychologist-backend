package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/internal/models"
	"github.com/pokonti/psychologist-backend/auth-service/internal/utils"
	"github.com/pokonti/psychologist-backend/auth-service/middleware"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required"` // "student", "psychologist"
}

type VerifyInput struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthController struct {
	UserClient userprofile.UserProfileServiceClient
}

//func (ac *AuthController) Register(c *gin.Context) {
//	var input RegisterInput
//
//	if err := c.ShouldBindJSON(&input); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//
//	hashedPassword, err := utils.HashPassword(input.Password)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
//		return
//	}
//
//	// Generate UUID for user
//	userID := uuid.NewString()
//	// Generate Code
//	verificationCode := utils.GenerateRandomCode()
//
//	user := models.User{
//		ID:               userID,
//		Email:            input.Email,
//		Password:         hashedPassword,
//		Role:             input.Role,
//		VerificationCode: verificationCode,
//		CodeExpiresAt:    time.Now().Add(15 * time.Minute), // Code valid for 15 mins
//		IsVerified:       false,
//	}
//
//	// Save in auth DB
//	if err := config.DB.Create(&user).Error; err != nil {
//		if errors.Is(err, gorm.ErrDuplicatedKey) {
//			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
//			return
//		}
//		log.Printf("Failed to create user in DB: %v", err)
//
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
//		return
//	}
//
//	go func() {
//		if err := utils.SendVerificationEmail(user.Email, verificationCode); err != nil {
//			log.Printf("Failed to send email to %s: %v", user.Email, err)
//		}
//	}()
//
//	// Call user-service via gRPC to create profile
//	_, err = ac.UserClient.CreateUserProfile(c.Request.Context(),
//		&userprofile.CreateUserProfileRequest{
//			Id:    user.ID,
//			Email: user.Email,
//			Role:  user.Role,
//		},
//	)
//
//	if err != nil {
//		// If gRPC fails, deleting user from Auth DB so they can try again
//		config.DB.Unscoped().Delete(&user)
//
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user profile, please try again"})
//		return
//	}
//
//	// Generate JWT with userID + email + role
//	token, err := middleware.GenerateJWT(user.ID, user.Email, user.Role)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
//		return
//	}
//
//	c.JSON(http.StatusCreated, gin.H{
//		"id":      user.ID,
//		"email":   user.Email,
//		"role":    user.Role,
//		"token":   token,
//		"message": "Registration successful. Please check your email for verification code.",
//	})
//}

func (ac *AuthController) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Check if user already exists
	var existingUser models.User
	result := config.DB.Where("email = ?", input.Email).First(&existingUser)

	if result.Error == nil {
		// User Found
		if existingUser.IsVerified {
			c.JSON(http.StatusConflict, gin.H{"error": "User already registered and verified. Please login."})
			return
		}

		// User exists but NOT verified: Resend Code
		newCode := utils.GenerateRandomCode()
		existingUser.VerificationCode = newCode
		existingUser.CodeExpiresAt = time.Now().Add(15 * time.Minute)

		// Update DB
		if err := config.DB.Save(&existingUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update verification code"})
			return
		}

		// Send Email
		go func() {
			utils.SendVerificationEmail(existingUser.Email, newCode)
		}()

		c.JSON(http.StatusOK, gin.H{"message": "User already exists but not verified. Verification code resent."})
		return
	}

	// 2. New User Logic
	hashedPassword, _ := utils.HashPassword(input.Password)
	verificationCode := utils.GenerateRandomCode()
	userID := uuid.NewString()

	newUser := models.User{
		ID:               userID,
		Email:            input.Email,
		Password:         hashedPassword,
		Role:             input.Role,
		VerificationCode: verificationCode,
		CodeExpiresAt:    time.Now().Add(15 * time.Minute),
		IsVerified:       false,
	}

	// Start Transaction
	tx := config.DB.Begin()

	// A. Save User to DB
	if err := tx.Create(&newUser).Error; err != nil {
		tx.Rollback()
		log.Printf("DB Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	// B. Send Email SYNCHRONOUSLY here to ensure it works before committing
	// If email fails, we want to rollback so the user isn't stuck in DB without a code
	if err := utils.SendVerificationEmail(newUser.Email, verificationCode); err != nil {
		tx.Rollback() // Delete user from DB if email fails
		log.Printf("Email Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email. Please try again."})
		return
	}

	// C. Call User Service (Create Profile)
	// We do this inside the transaction too, or right before commit
	_, err := ac.UserClient.CreateUserProfile(c.Request.Context(), &userprofile.CreateUserProfileRequest{
		Id:    newUser.ID,
		Email: newUser.Email,
		Role:  newUser.Role,
	})
	if err != nil {
		tx.Rollback() // Rollback Auth DB if User Service fails
		log.Printf("gRPC Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user profile"})
		return
	}

	// Commit Transaction
	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email for verification code.",
	})
}

func (ac *AuthController) VerifyEmail(c *gin.Context) {
	var input VerifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already verified"})
		return
	}

	if time.Now().After(user.CodeExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification code expired"})
		return
	}

	if user.VerificationCode != input.Code {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification code"})
		return
	}

	// Update User
	user.IsVerified = true
	user.VerificationCode = "" // Clear code
	config.DB.Save(&user)

	token, _ := middleware.GenerateJWT(user.ID, user.Email, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
		"token":   token,
	})
}

func (ac *AuthController) Login(c *gin.Context) {
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

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Please verify your email first"})
		return
	}

	token, err := middleware.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
