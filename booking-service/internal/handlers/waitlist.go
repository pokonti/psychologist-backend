package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

// JoinWaitlist godoc
// @Summary      Join a waitlist
// @Description  Student joins a waitlist for a specific psychologist on a specific date.
// @Tags         student-waitlist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.JoinWaitlistInput true "Waitlist details"
// @Success      201 {object} models.MessageResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      409 {object} models.ErrorResponse "Already on waitlist"
// @Router       /student/waitlist [post]
func (h *BookingHandler) JoinWaitlist(c *gin.Context) {
	studentID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "student" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only students can join waitlists"})
		return
	}

	var input models.JoinWaitlistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Basic date validation
	if _, err := parseDate(input.Date); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	entry := models.WaitlistEntry{
		ID:             uuid.NewString(),
		StudentID:      studentID,
		PsychologistID: input.PsychologistID,
		Date:           input.Date,
	}

	// Save to DB. If it violates the unique constraint, it means they already joined.
	if err := config.DB.Create(&entry).Error; err != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "You are already on the waitlist for this date and psychologist"})
		return
	}

	c.JSON(http.StatusCreated, models.MessageResponse{Message: "Successfully joined the waitlist"})
}

// GetMyWaitlist godoc
// @Summary      View my waitlists
// @Description  Student sees all the dates and psychologists they are waiting for.
// @Tags         student-waitlist
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.WaitlistResponse
// @Router       /student/waitlist [get]
func (h *BookingHandler) GetMyWaitlist(c *gin.Context) {
	studentID := c.GetHeader("X-User-ID")

	var entries []models.WaitlistEntry
	if err := config.DB.Where("student_id = ?", studentID).Order("date asc").Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	// Enrich with Psychologist Names via gRPC
	var psychIDs []string
	uniquePsychs := make(map[string]bool)
	for _, e := range entries {
		if !uniquePsychs[e.PsychologistID] {
			uniquePsychs[e.PsychologistID] = true
			psychIDs = append(psychIDs, e.PsychologistID)
		}
	}

	psychMap := make(map[string]string)
	if len(psychIDs) > 0 {
		grpcResp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{Ids: psychIDs})
		if err == nil {
			for _, p := range grpcResp.Profiles {
				psychMap[p.Id] = p.FullName
			}
		} else {
			log.Printf("Failed to fetch psychologist profiles for waitlist: %v", err)
		}
	}

	var response []models.WaitlistResponse
	for _, e := range entries {
		name := "Unknown Specialist"
		if val, ok := psychMap[e.PsychologistID]; ok {
			name = val
		}
		response = append(response, models.WaitlistResponse{
			ID:               e.ID,
			PsychologistID:   e.PsychologistID,
			PsychologistName: name,
			Date:             e.Date,
			CreatedAt:        e.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

// LeaveWaitlist godoc
// @Summary      Leave a waitlist
// @Description  Student removes themselves from a specific waitlist entry.
// @Tags         student-waitlist
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Waitlist Entry ID"
// @Success      200 {object} models.MessageResponse
// @Router       /student/waitlist/{id} [delete]
func (h *BookingHandler) LeaveWaitlist(c *gin.Context) {
	entryID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")

	// Delete where ID matches AND StudentID matches
	res := config.DB.Where("id = ? AND student_id = ?", entryID, studentID).Delete(&models.WaitlistEntry{})

	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Waitlist entry not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Removed from waitlist"})
}
