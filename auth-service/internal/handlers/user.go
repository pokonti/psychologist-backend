package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/internal/clients"
	"github.com/pokonti/psychologist-backend/auth-service/internal/models"
	"github.com/pokonti/psychologist-backend/auth-service/internal/utils"
	"github.com/pokonti/psychologist-backend/auth-service/middleware"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

type AuthController struct {
	UserClient userprofile.UserProfileServiceClient
	RabbitMQ   *clients.RabbitMQClient
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user in Auth DB, sends verification email, and creates profile in User Service
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body models.RegisterInput true "User Registration Info"
// @Success      201  {object}  models.MessageResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      409  {object}  models.ErrorResponse "User already exists"
// @Failure      500  {object}  models.ErrorResponse
// @Router       /auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var input models.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	result := config.DB.Where("email = ?", input.Email).First(&existingUser)

	if result.Error == nil {
		// User Found
		if existingUser.IsVerified {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "User already registered and verified. Please login."})
			return
		}

		// User exists but NOT verified: Resend Code
		newCode := utils.GenerateRandomCode()
		existingUser.VerificationCode = newCode
		existingUser.CodeExpiresAt = time.Now().Add(15 * time.Minute)

		// Update DB
		if err := config.DB.Save(&existingUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update verification code"})
			return
		}

		// Send Email
		msg := clients.NotificationMessage{
			Type:    "auth_verification",
			ToEmail: existingUser.Email,
			Data: map[string]string{
				"code": newCode,
			},
		}
		go func() { ac.RabbitMQ.PublishNotification(msg) }()

		c.JSON(http.StatusOK, models.MessageResponse{Message: "User already exists but not verified. Verification code resent."})
		return
	}

	// New User Logic
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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to register user"})
		return
	}

	// B. Publish to RabbitMQ
	// We don't rollback if this fails here, because RabbitMQ itself is the "safe" place
	msg := clients.NotificationMessage{
		Type:    "auth_verification",
		ToEmail: newUser.Email,
		Data: map[string]string{
			"code": verificationCode,
		},
	}

	// We publish the message. If publishing fails, we CAN rollback.
	if err := ac.RabbitMQ.PublishNotification(msg); err != nil {
		tx.Rollback()
		log.Printf("RabbitMQ Error: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to queue verification email"})
		return
	}

	// C. Call User Service
	// We do this inside the transaction too, or right before commit
	_, err := ac.UserClient.CreateUserProfile(c.Request.Context(), &userprofile.CreateUserProfileRequest{
		Id:    newUser.ID,
		Email: newUser.Email,
		Role:  newUser.Role,
	})
	if err != nil {
		tx.Rollback() // Rollback Auth DB if User Service fails
		log.Printf("gRPC Error: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create user profile"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, models.MessageResponse{Message: "Registration successful. Please check your email for verification code."})
}

// VerifyEmail godoc
// @Summary      Verify Email
// @Description  Verifies the 6-digit code sent to email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body models.VerifyInput true "Verification Code"
// @Success      200  {object}  models.TokenResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      401  {object}  models.ErrorResponse "Invalid Code"
// @Router       /auth/verify [post]
func (ac *AuthController) VerifyEmail(c *gin.Context) {
	var input models.VerifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "User already verified"})
		return
	}

	if time.Now().After(user.CodeExpiresAt) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Verification code expired"})
		return
	}

	if user.VerificationCode != input.Code {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid verification code"})
		return
	}

	// Update User
	user.IsVerified = true
	user.VerificationCode = ""
	config.DB.Save(&user)

	token, _ := middleware.GenerateJWT(user.ID, user.Email, user.Role)

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
		"token":   token,
	})
}

// Login godoc
// @Summary      Login User
// @Description  Authenticates user and returns JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body models.LoginInput true "Login Credentials"
// @Success      200  {object}  models.TokenResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      401  {object}  models.ErrorResponse
// @Router       /auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var input models.LoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid credentials"})
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid credentials"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Please verify your email first"})
		return
	}

	if user.IsBlocked {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Your account has been blocked by the administrator."})
		return
	}

	token, err := middleware.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	refreshToken := uuid.NewString()

	config.DB.Model(&user).Update("refresh_token", refreshToken)

	c.JSON(http.StatusOK, models.TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
	})
}

// RefreshToken godoc
// @Summary      Get a new access token
// @Description  Uses a valid refresh token to generate a new 15-minute access token. Implements Refresh Token Rotation (returns a new refresh token too).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.RefreshInput true "Refresh Token"
// @Success      200 {object} models.TokenResponse
// @Failure      401 {object} models.ErrorResponse "Invalid or expired refresh token"
// @Router       /auth/refresh [post]
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var input models.RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Find user by Refresh Token
	var user models.User
	if err := config.DB.Where("refresh_token = ?", input.RefreshToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid refresh token. Please log in again."})
		return
	}

	if user.IsBlocked {
		config.DB.Model(&user).Update("refresh_token", "")
		c.JSON(http.StatusForbidden, gin.H{"error": "Account blocked. Access revoked."})
		return
	}

	newAccessToken, _ := middleware.GenerateJWT(user.ID, user.Email, user.Role)

	newRefreshToken := uuid.NewString()
	config.DB.Model(&user).Update("refresh_token", newRefreshToken)

	c.JSON(http.StatusOK, models.TokenResponse{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
	})
}

// Logout godoc
// @Summary      Logout user
// @Description  Revokes the refresh token, forcing the user to log in again once their access token expires.
// @Tags         auth
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")

	config.DB.Model(&models.User{}).Where("id = ?", userID).Update("refresh_token", "")

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Successfully logged out"})
}
