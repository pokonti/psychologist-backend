package handlers

import (
	"net/http"

	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/user-service/internal/repository"
)

type ProfileHandler struct {
	Repo repository.ProfileRepository
}

func NewProfileHandler(repo repository.ProfileRepository) *ProfileHandler {
	return &ProfileHandler{Repo: repo}
}

// GetMyProfile godoc
// @Summary      Get current user's profile
// @Description  Returns the profile of the currently authenticated user. In production, the gateway injects X-User-ID based on the JWT. When calling the service directly (e.g. via Swagger), you must provide X-User-ID manually.
// @Tags         profile
// @Produce      json
// @Param        X-User-ID  header    string  true  "User ID (UUID from JWT sub)"
// @Success      200  {object}  models.UserProfile
// @Failure      401  {object}  map[string]string  "missing user id"
// @Failure      404  {object}  map[string]string  "profile not found"
// @Failure      500  {object}  map[string]string  "internal error"
// @Router       /users/me [get]
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	profile, err := h.Repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

type UpdateProfileRequest struct {
	FullName            *string `json:"full_name" binding:"omitempty"`
	Gender              *string `json:"gender" binding:"omitempty"`
	Specialization      *string `json:"specialization" binding:"omitempty"`
	Bio                 *string `json:"bio" binding:"omitempty"`
	AvatarURL           *string `json:"avatar_url" binding:"omitempty"`
	Phone               *string `json:"phone" binding:"omitempty"`
	NotificationChannel *string `json:"notification_channel" binding:"omitempty"`
}

// UpdateMyProfile godoc
// @Summary      Update current user's profile
// @Description  Partially update the profile of the current user. In production, the gateway injects X-User-ID from JWT. When calling user-service directly (Swagger), provide X-User-ID manually.
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        X-User-ID  header    string               true  "User ID (UUID from JWT sub)"
// @Param        request    body      UpdateProfileRequest true  "Fields to update"
// @Success      200        {object}  models.UserProfile
// @Failure      400        {object}  map[string]string  "invalid request body"
// @Failure      401        {object}  map[string]string  "missing user id"
// @Failure      500        {object}  map[string]string  "failed to update profile"
// @Router       /users/me [put]
func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.Repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Option: create profile automatically if not exists
			profile = &models.UserProfile{ID: userID}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
	}
	// Patch fields if provided
	if req.FullName != nil {
		profile.FullName = *req.FullName
	}
	if req.Gender != nil {
		profile.Gender = *req.Gender
	}
	if req.Specialization != nil {
		profile.Specialization = *req.Specialization
	}
	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = *req.AvatarURL
	}
	if req.Phone != nil {
		profile.Phone = *req.Phone
	}
	if req.NotificationChannel != nil {
		profile.NotificationChannel = *req.NotificationChannel
	}

	if err := h.Repo.Update(c.Request.Context(), profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// GetAllPsychologists godoc
// @Summary      List all psychologists
// @Description  Returns a list of all users whose role is 'psychologist'.
// @Tags         psychologists
// @Produce      json
// @Success      200  {array}   models.UserProfile
// @Failure      500  {object}  map[string]string  "database error"
// @Security     BearerAuth
// @Router       /psychologists [get]
func (h *ProfileHandler) GetAllPsychologists(c *gin.Context) {
	profiles, err := h.Repo.GetAllPsychologists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, profiles)
}
