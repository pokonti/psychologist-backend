package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
	"gorm.io/gorm/clause"
)

// CreateSlot godoc
// @Summary      Create schedule slots for a psychologist
// @Description  Psychologist generates multiple slots based on a weekly schedule pattern between start_date and end_date.
// @Tags         psychologist-slots
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Param        request      body    models.CreateScheduleInput  true  "Schedule configuration"
// @Success      201 {object}   models.ScheduleCreatedResponse
// @Failure      400  {object}  models.ErrorResponse              "validation error or no slots created"
// @Failure      403  {object}  models.ErrorResponse              "only psychologists can create slots"
// @Failure      500  {object}  models.ErrorResponse              "database error"
// @Router       /psychologist/slots [post]
func CreateSlot(c *gin.Context) {
	psychologistID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only psychologists can create slots"})
		return
	}

	var input models.CreateScheduleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default duration
	if input.Duration == 0 {
		input.Duration = 50
	}

	// 1. Convert Schedule Slice to a Map for faster lookup
	// Map Key: DayOfWeek (int), Value: Slice of Time Strings
	scheduleMap := make(map[int][]string)
	for _, day := range input.Schedule {
		scheduleMap[day.DayOfWeek] = day.StartTimes
	}

	// 2. Parse Start/End Dates
	currentDate, err := parseDate(input.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid start_date",
		})
		return
	}
	endDate, err := parseDate(input.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid end_date",
		})
		return
	}

	var slotsToCreate []models.Slot

	// 3. Loop through every calendar date
	for !currentDate.After(endDate) {

		// Get the weekday of the current iteration (0=Sun, 1=Mon...)
		currentWeekday := int(currentDate.Weekday())

		// Check if we have times defined for this weekday
		if times, exists := scheduleMap[currentWeekday]; exists {

			// Loop through the specific times (09:00, 10:00, etc.)
			for _, timeStr := range times {

				// Combine Date + Time
				slotTime, err := combineDateAndTime(currentDate, timeStr)
				if err != nil {
					continue // Skip invalid time formats
				}

				slotsToCreate = append(slotsToCreate, models.Slot{
					ID:             uuid.NewString(),
					PsychologistID: psychologistID,
					StartTime:      slotTime,
					Duration:       input.Duration,
					IsBooked:       false,
					Version:        1,
				})
			}
		}

		// Move to next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	if len(slotsToCreate) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "No slots created. Check your dates and schedule.",
		})
		return
	}

	// Batch Insert with "ON CONFLICT DO NOTHING"
	// if a slot already exists for this psych at this time, skip it
	err = config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&slotsToCreate).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create slots",
		})
		return
	}

	c.JSON(http.StatusCreated, models.ScheduleCreatedResponse{
		Message: "Schedule created successfully",
		Count:   len(slotsToCreate),
	})
}

// GetMySchedule godoc
// @Summary      Get psychologist's own schedule
// @Description  Returns all slots for the logged-in psychologist, including student details for booked slots.
// @Tags         psychologist-slots
// @Produce      json
// @Security     BearerAuth
// @Param        date   query     string  false  "Filter by date (YYYY-MM-DD)"
// @Success      200    {array}   models.PsychologistScheduleResponse
// @Router       /psychologist/slots [get]
func (h *BookingHandler) GetMySchedule(c *gin.Context) {
	psychID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only psychologists can access this"})
		return
	}

	// Optional Date Filter
	dateStr := c.Query("date")

	query := config.DB.Where("psychologist_id = ?", psychID).Order("start_time asc")

	if dateStr != "" {
		date, err := parseDate(dateStr)
		if err == nil {
			nextDay := date.Add(24 * time.Hour)
			query = query.Where("start_time >= ? AND start_time < ?", date, nextDay)
		}
	}

	var slots []models.Slot
	if err := query.Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Collect Student IDs for gRPC
	var studentIDs []string
	for _, s := range slots {
		if s.IsBooked && s.StudentID != nil {
			studentIDs = append(studentIDs, *s.StudentID)
		}
	}

	// Batch Fetch Student Profiles
	studentMap := make(map[string]string)
	if len(studentIDs) > 0 {
		grpcResp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{
			Ids: studentIDs,
		})
		if err == nil {
			for _, p := range grpcResp.Profiles {
				studentMap[p.Id] = p.FullName
			}
		} else {
			log.Printf("Failed to fetch student profiles: %v", err)
		}
	}

	var response []models.PsychologistScheduleResponse
	for _, s := range slots {
		studentName := ""
		if s.StudentID != nil {
			if name, ok := studentMap[*s.StudentID]; ok {
				studentName = name
			} else {
				studentName = "Unknown Student"
			}
		}

		response = append(response, models.PsychologistScheduleResponse{
			ID:                   s.ID,
			StartTime:            s.StartTime,
			Duration:             s.Duration,
			IsBooked:             s.IsBooked,
			BookingType:          s.BookingType,
			PsychologistID:       s.PsychologistID,
			StudentID:            s.StudentID,
			StudentName:          studentName,
			QuestionnaireAnswers: s.QuestionnaireAnswers,
		})
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSlot godoc
// @Summary      Delete an unbooked slot
// @Description  Allows a psychologist to remove an available slot from their schedule. Fails if the slot is already booked.
// @Tags         psychologist-slots
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Slot ID (UUID)"
// @Success      200  {object}  models.MessageResponse
// @Failure      400  {object}  models.ErrorResponse "Invalid request"
// @Failure      403  {object}  models.ErrorResponse "Not authorized or not the owner"
// @Failure      404  {object}  models.ErrorResponse "Slot not found"
// @Failure      409  {object}  models.ErrorResponse "Cannot delete a booked slot"
// @Failure      500  {object}  models.ErrorResponse "Database error"
// @Router       /psychologist/slots/{id} [delete]
func (h *BookingHandler) DeleteSlot(c *gin.Context) {
	slotID := c.Param("id")
	psychID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Only psychologists can delete slots",
		})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Slot not found",
		})
		return
	}

	if slot.PsychologistID != psychID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "You can only delete your own slots",
		})
		return
	}

	// Prevent deleting a slot if a student has already booked it.
	// (Canceling a booked appointment will be a separate feature that notifies the student).
	if slot.IsBooked {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Cannot delete this slot because it is already booked by a student. Please use the Cancel Appointment feature.",
		})
		return
	}

	// Proceed with deletion
	if err := config.DB.Delete(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to delete slot",
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Slot successfully deleted",
	})
}
