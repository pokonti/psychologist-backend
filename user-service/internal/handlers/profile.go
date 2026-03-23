package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// LogMood godoc
// @Summary      Log daily mood
// @Description  Student selects how they are feeling today. Updates the existing entry if already logged today.
// @Tags         well-being
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.LogMoodInput true "Mood selection"
// @Success      200 {object} map[string]string
// @Router       /users/me/mood [post]
func (h *ProfileHandler) LogMood(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")

	var input models.LogMoodInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	today := time.Now().Format("2006-01-02")
	score := getMoodScore(input.Mood)

	moodLog := models.MoodLog{
		ID:     uuid.NewString(),
		UserID: userID,
		Date:   today,
		Mood:   input.Mood,
		Score:  score,
	}

	err := config.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"mood", "score", "updated_at"}),
	}).Create(&moodLog).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to log mood"})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Mood logged successfully!",
	})
}

// GetMoodGraphic godoc
// @Summary      Get mood data for graphic
// @Description  Returns mood scores for the graph. Filter by 'last_week' or 'last_month'.
// @Tags         well-being
// @Produce      json
// @Security     BearerAuth
// @Param        filter query string false "Filter period (default: last_week)" Enums(last_week, last_month)
// @Success      200 {array} models.MoodGraphicResponse
// @Router       /users/me/mood/graphic [get]
func (h *ProfileHandler) GetMoodGraphic(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	filter := c.DefaultQuery("filter", "last_week")

	now := time.Now()
	var startDate time.Time

	if filter == "last_month" {
		startDate = now.AddDate(0, -1, 0)
	} else {
		startDate = now.AddDate(0, 0, -6)
	}

	var logs []models.MoodLog
	config.DB.
		Where("user_id = ? AND date >= ?", userID, startDate.Format("2006-01-02")).
		Order("date asc").
		Find(&logs)

	var response []models.MoodGraphicResponse
	for _, log := range logs {
		parsedDate, _ := time.Parse("2006-01-02", log.Date)
		response = append(response, models.MoodGraphicResponse{
			Date:      log.Date,
			DayOfWeek: parsedDate.Format("Mon"), // Returns "Mon", "Tue", etc.
			Mood:      log.Mood,
			Score:     log.Score,
		})
	}

	c.JSON(http.StatusOK, response)
}

func getMoodScore(mood string) int {
	switch mood {
	case "Amazing":
		return 6
	case "Nice":
		return 5
	case "Not bad":
		return 4
	case "Sad":
		return 3
	case "Anxiously":
		return 2
	case "Stressed":
		return 1
	default:
		return 0
	}
}
