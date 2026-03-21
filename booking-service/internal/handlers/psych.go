package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/booking-service/clients"
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
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only psychologists can create slots"})
		return
	}

	var input models.CreateScheduleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	if input.Duration == 0 {
		input.Duration = 50
	}

	scheduleMap := make(map[int][]string)
	for _, day := range input.Schedule {
		scheduleMap[day.DayOfWeek] = day.StartTimes
	}
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
					Status:         models.StatusAvailable,
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
// @Failure      401 {object} models.ErrorResponse
// @Router       /psychologist/slots [get]
func (h *BookingHandler) GetMySchedule(c *gin.Context) {
	psychID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only psychologists can access this"})
		return
	}

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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	var studentIDs []string
	for _, s := range slots {
		if s.Status != models.StatusAvailable && s.StudentID != nil {
			studentIDs = append(studentIDs, *s.StudentID)
		}
	}

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
			Status:               s.Status,
			BookingType:          s.BookingType,
			PsychologistID:       s.PsychologistID,
			StudentID:            s.StudentID,
			StudentName:          studentName,
			QuestionnaireAnswers: s.QuestionnaireAnswers,
			PhoneNumber:          s.PhoneNumber,
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
	if slot.Status != models.StatusAvailable {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Cannot delete a slot that is reserved or booked",
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

// AddSessionNote godoc
// @Summary      Add private notes to a session
// @Description  Psychologist writes private notes after a session. Hidden from the student.
// @Tags         psychologist-slots
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path   string        true  "Slot ID"
// @Param        request body   models.AddNoteInput  true  "The private notes"
// @Success      200 {object} models.MessageResponse
// @Router       /psychologist/slots/{id}/notes [put]
func (h *BookingHandler) AddSessionNote(c *gin.Context) {
	slotID := c.Param("id")
	psychID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only psychologists can add notes"})
		return
	}

	var input models.AddNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Slot not found"})
		return
	}

	if slot.PsychologistID != psychID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "You can only add notes to your own sessions"})
		return
	}
	if slot.Status != models.StatusBooked {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Cannot add notes to an unbooked slot"})
		return
	}

	slot.PsychologistNotes = input.Notes
	if err := config.DB.Save(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to save notes"})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Notes saved successfully"})
}

// GetStudentHistory godoc
// @Summary      Get a student's session history
// @Description  Psychologist views all past sessions and private notes for a specific student.
// @Tags         psychologist-slots
// @Produce      json
// @Security     BearerAuth
// @Param        student_id path string true "Student ID"
// @Success      200 {array} models.StudentHistoryResponse
// @Router       /psychologist/students/{student_id}/history [get]
func (h *BookingHandler) GetStudentHistory(c *gin.Context) {
	studentID := c.Param("student_id")
	psychID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Access denied"})
		return
	}

	// Fetch all past bookings between THIS psychologist and THIS student
	var slots []models.Slot
	if err := config.DB.
		Where("psychologist_id = ? AND student_id = ? AND status = ?", psychID, studentID, models.StatusBooked).
		Order("start_time desc").
		Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	var history []models.StudentHistoryResponse
	for _, s := range slots {
		history = append(history, models.StudentHistoryResponse{
			SlotID:               s.ID,
			StartTime:            s.StartTime,
			BookingType:          s.BookingType,
			QuestionnaireAnswers: s.QuestionnaireAnswers,
			PsychologistNotes:    s.PsychologistNotes,
		})
	}

	if len(history) == 0 {
		c.JSON(http.StatusOK, []models.StudentHistoryResponse{})
		return
	}

	c.JSON(http.StatusOK, history)
}

// CancelBookingByPsychologist godoc
// @Summary      Psychologist cancels a booked appointment
// @Description  Psychologist cancels a session. Frees the slot and triggers a cancellation email to the student.
// @Tags         psychologist-slots
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Slot ID (UUID)"
// @Success      200  {object}  models.MessageResponse
// @Failure      403  {object}  models.ErrorResponse "Not authorized"
// @Failure      404  {object}  models.ErrorResponse "Slot not found"
// @Failure      409  {object}  models.ErrorResponse "Slot is not booked"
// @Router       /psychologist/slots/{id}/cancel [post]
func (h *BookingHandler) CancelBookingByPsychologist(c *gin.Context) {
	slotID := c.Param("id")
	psychID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only psychologists can cancel appointments"})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Slot not found"})
		return
	}

	if slot.PsychologistID != psychID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "You can only cancel your own appointments"})
		return
	}

	if slot.Status != models.StatusBooked || slot.StudentID == nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "This slot is not booked"})
		return
	}

	studentID := *slot.StudentID

	// Atomically free the slot
	result := config.DB.Model(&models.Slot{}).
		Where("id = ? AND version = ?", slot.ID, slot.Version).
		Updates(map[string]interface{}{
			"status":                models.StatusAvailable,
			"student_id":            nil,
			"booking_type":          "",
			"questionnaire_answers": "",
			"version":               slot.Version + 1,
		})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Update failed, try again"})
		return
	}

	go func() {
		resp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{studentID, psychID},
		})

		var studentEmail, psychName string
		if err == nil {
			for _, p := range resp.Profiles {
				if p.Id == studentID {
					studentEmail = p.Email
				} else if p.Id == psychID {
					psychName = p.FullName
				}
			}
		}

		if studentEmail != "" {
			msg := clients.NotificationMessage{
				Type:    "booking_cancellation_by_psychologist",
				ToEmail: studentEmail,
				Data: map[string]string{
					"psychologist_name": psychName,
					"datetime":          slot.StartTime.Format("Monday, 02 Jan 2006 at 15:04"),
				},
			}
			h.RabbitMQ.PublishNotification(msg)
		}
	}()

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Appointment canceled and student notified"})
}

// AddRecommendation godoc
// @Summary      Add recommendations for a student
// @Description  Psychologist writes post-session recommendations. This is visible to the student and triggers an email notification.
// @Tags         psychologist-slots
// @Security     BearerAuth
// @Param        id       path      string                      true  "Slot ID (UUID of the completed session)"
// @Param        request  body      models.RecommendationInput  true  "The recommendations text"
// @Success      200      {object}  models.MessageResponse      "Recommendations successfully saved and student notified"
// @Failure      400      {object}  models.ErrorResponse        "Invalid request body"
// @Failure      403      {object}  models.ErrorResponse        "Not authorized or not the owner of this session"
// @Failure      404      {object}  models.ErrorResponse        "Slot not found"
// @Failure      500      {object}  models.ErrorResponse        "Database or internal server error"
// @Router       /psychologist/slots/{id}/recommendations [put]
func (h *BookingHandler) AddRecommendation(c *gin.Context) {
	slotID := c.Param("id")
	psychID := c.GetHeader("X-User-ID")

	if c.GetHeader("X-User-Role") != "psychologist" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only psychologists can do this"})
		return
	}

	var input models.RecommendationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Slot not found"})
		return
	}

	if slot.PsychologistID != psychID || slot.Status != models.StatusBooked || slot.StudentID == nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Cannot add recommendations to this slot"})
		return
	}

	slot.StudentRecommendations = input.Recommendations
	if err := config.DB.Save(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	go func() {
		resp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{*slot.StudentID, psychID},
		})

		var studentEmail, psychName string
		if err == nil {
			for _, p := range resp.Profiles {
				if p.Id == *slot.StudentID {
					studentEmail = p.Email
				}
				if p.Id == psychID {
					psychName = p.FullName
				}
			}
		}

		if studentEmail != "" {
			msg := clients.NotificationMessage{
				Type:    "new_recommendation",
				ToEmail: studentEmail,
				Data: map[string]string{
					"psychologist_name": psychName,
					"date":              slot.StartTime.Format("02 Jan 2006"),
				},
			}
			h.RabbitMQ.PublishNotification(msg)
		}
	}()

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Recommendations shared with student"})
}
