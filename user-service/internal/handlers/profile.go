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
// @Security     BearerAuth
// @Success      200        {object}  models.UserProfile
// @Failure      401        {object}  models.ErrorResponse  "missing user id"
// @Failure      404        {object}  models.ErrorResponse  "profile not found"
// @Failure      500        {object}  models.ErrorResponse  "internal error"
// @Router       /users/me [get]
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Missing user id",
		})
		return
	}

	profile, err := h.Repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: "Profile not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
		})
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
// @Security     BearerAuth
// @Param        request    body      UpdateProfileRequest true  "Fields to update"
// @Success      200        {object}  models.UserProfile
// @Failure      400        {object}  models.ErrorResponse "invalid request body"
// @Failure      401        {object}  models.ErrorResponse "missing user id"
// @Failure      500        {object}  models.ErrorResponse "failed to update profile"
// @Router       /users/me [put]
func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Missing user id",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	profile, err := h.Repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			profile = &models.UserProfile{ID: userID}
		} else {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: err.Error(),
			})
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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to update profile:" + err.Error(),
		})
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
// @Failure      500  {object}  models.ErrorResponse "database error"
// @Security     BearerAuth
// @Router       /users/psychologists [get]
func (h *ProfileHandler) GetAllPsychologists(c *gin.Context) {
	profiles, err := h.Repo.GetAllPsychologists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Database error:" + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, profiles)
}
