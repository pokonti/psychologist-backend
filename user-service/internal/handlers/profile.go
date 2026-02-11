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

// GET /users/me
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

// PUT /users/me
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
