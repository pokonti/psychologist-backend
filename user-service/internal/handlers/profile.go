package handlers

import (
	"errors"
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

// UpdateMyProfile godoc
// @Summary      Update current user's profile
// @Description  Partially update the profile of the current user. In production, the gateway injects X-User-ID from JWT. When calling user-service directly (Swagger), provide X-User-ID manually.
// @Tags         profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request    body      models.UpdateProfileRequest true  "Fields to update"
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

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	profile, err := h.Repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

// GetPublicPsychologists godoc
// @Summary      List psychologists for booking
// @Description  Returns a sanitized list of psychologists for students to browse. Hides sensitive data.
// @Tags         users
// @Produce      json
// @Success      200  {array}   models.PublicPsychologistResponse
// @Failure      500  {object}  models.ErrorResponse "database error"
// @Security     BearerAuth
// @Router       /users/psychologists [get]
func (h *ProfileHandler) GetPublicPsychologists(c *gin.Context) {
	profiles, err := h.Repo.GetAllPsychologists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Database error:" + err.Error(),
		})
		return
	}

	var publicProfiles []models.PublicPsychologistResponse
	for _, p := range profiles {
		publicProfiles = append(publicProfiles, models.PublicPsychologistResponse{
			ID:             p.ID,
			FullName:       p.FullName,
			Gender:         p.Gender,
			Bio:            p.Bio,
			Specialization: p.Specialization,
			AvatarURL:      p.AvatarURL,
			Experience:     p.Experience,
			Description:    p.Description,
			Rating:         p.Rating,
		})
	}

	if len(publicProfiles) == 0 {
		c.JSON(http.StatusOK, []models.PublicPsychologistResponse{})
		return
	}

	c.JSON(http.StatusOK, publicProfiles)
}
