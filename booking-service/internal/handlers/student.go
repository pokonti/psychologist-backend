package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/clients"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

// GET /slots/calendar?psychologist_id=...&year=2026&month=2

// GetCalendarAvailability godoc
// @Summary      Get calendar days with available slots
// @Description  Returns dates within the specified month where the psychologist has at least one free slot.
// @Tags         slots
// @Produce      json
// @Security BearerAuth
// @Param        psychologist_id  query   string  true  "Psychologist ID"
// @Param        year             query   string  true  "Year (e.g. 2026)"
// @Param        month            query   string  true  "Month (1-12 or 01-12)"
// @Success      200 {object}     models.CalendarAvailabilityResponse
// @Failure      400  {object}    models.ErrorResponse "missing params"
// @Router       /slots/calendar [get]
func (h *BookingHandler) GetCalendarAvailability(c *gin.Context) {
	psychID := c.Query("psychologist_id")
	year := c.Query("year")
	month := c.Query("month")

	if psychID == "" || year == "" || month == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Missing params",
		})
		return
	}

	startDate := fmt.Sprintf("%s-%s-01", year, month)
	var availableDays []string

	// Postgres Query to find days with at least 1 free slot
	config.DB.Model(&models.Slot{}).
		Select("TO_CHAR(start_time, 'YYYY-MM-DD')").
		Where("psychologist_id = ?", psychID).
		Where("is_booked = ?", false).
		Where("start_time >= ? AND start_time < (?::date + '1 month'::interval)", startDate, startDate).
		Group("TO_CHAR(start_time, 'YYYY-MM-DD')").
		Find(&availableDays)

	c.JSON(http.StatusOK, models.CalendarAvailabilityResponse{
		AvailableDates: availableDays,
	})
}

// GET /slots?psychologist_id=...&date=2026-02-20

// GetAvailableSlots godoc
// @Summary      Get available slots for a specific date
// @Description  Returns free slots for a given psychologist and date, enriched with psychologist name via gRPC.
// @Tags         slots
// @Produce      json
// @Security BearerAuth
// @Param        psychologist_id  query   string  true  "Psychologist ID"
// @Param        date             query   string  true  "Date in format YYYY-MM-DD"
// @Success      200 {array} models.SlotResponse
// @Failure      400  {object}  models.ErrorResponse "missing params"
// @Failure      500  {object}  models.ErrorResponse "database or gRPC error"
// @Router       /slots [get]
func (h *BookingHandler) GetAvailableSlots(c *gin.Context) {
	psychID := c.Query("psychologist_id")
	dateStr := c.Query("date")

	if psychID == "" || dateStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Missing params",
		})
		return
	}

	date, _ := parseDate(dateStr)
	nextDay := date.Add(24 * time.Hour)

	var slots []models.Slot
	if err := config.DB.
		Where("psychologist_id = ?", psychID).
		Where("is_booked = ?", false).
		Where("start_time >= ? AND start_time < ?", date, nextDay).
		Order("start_time asc").
		Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "DB Error",
		})
		return
	}

	// Collect ID
	psychIDList := []string{psychID} // In this case, we only have one ID because of the filter

	// Call User Service
	grpcResp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{
		Ids: psychIDList,
	})

	psychName := "-"
	if err == nil && len(grpcResp.Profiles) > 0 {
		psychName = grpcResp.Profiles[0].FullName
	}

	// Build Response
	var response []models.SlotResponse
	for _, s := range slots {
		response = append(response, models.SlotResponse{
			ID:               s.ID,
			StartTime:        s.StartTime,
			Duration:         s.Duration,
			IsBooked:         s.IsBooked,
			PsychologistID:   s.PsychologistID,
			PsychologistName: psychName,
		})
	}

	c.JSON(http.StatusOK, response)
}

// BookSlot godoc
// @Summary      Book a slot as a student
// @Description  Student books a specific slot. Uses optimistic locking on the version field to avoid double-booking.
// @Tags         student-booking
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Param        id         path    string         true  "Slot ID"
// @Param        request  body   models.BookSlotInput  true  "Booking details: type and answers"
// @Success 200 {object} models.MessageResponse
// @Failure      400  {object}  models.ErrorResponse "invalid request body"
// @Failure      404  {object}  models.ErrorResponse "slot not found"
// @Failure      409  {object}  models.ErrorResponse "slot already booked or just booked"
// @Failure      500  {object}  models.ErrorResponse "database error"
// @Router       /student/slots/{id}/book [post]
func (h *BookingHandler) BookSlot(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")

	var input models.BookSlotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Slot not found",
		})
		return
	}

	if slot.IsBooked {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Slot is already booked",
		})
		return
	}

	// OPTIMISTIC UPDATE
	// UPDATE slots SET is_booked=true, student_id='...', version=2
	// WHERE id='...' AND version=1
	// We increment version manually to invalidate other requests
	result := config.DB.Model(&models.Slot{}).
		Where("id = ? AND version = ?", slot.ID, slot.Version).
		Updates(map[string]interface{}{
			"is_booked":             true,
			"student_id":            studentID,
			"booking_type":          input.BookingType,
			"questionnaire_answers": input.Answers,
			"version":               slot.Version + 1,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Database error",
		})
		return
	}

	// Check if the row was actually touched
	if result.RowsAffected == 0 {
		// If 0 rows were updated, it means the Version changed between
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Slot was just booked by someone else",
		})
		return
	}

	go func() {
		grpcResp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{studentID, slot.PsychologistID},
		})

		var studentEmail, psychName string
		if err == nil {
			for _, p := range grpcResp.Profiles {
				if p.Id == studentID {
					studentEmail = p.Email
				} else if p.Id == slot.PsychologistID {
					psychName = p.FullName
				}
			}
		}

		formattedDate := slot.StartTime.Format("Monday, 02 Jan 2006 at 15:04")

		msg := clients.NotificationMessage{
			Type:    "booking_confirmation",
			ToEmail: studentEmail,
			Data: map[string]string{
				"psychologist_name": psychName,
				"datetime":          formattedDate,
				"format":            input.BookingType,
			},
		}

		h.RabbitMQ.PublishNotification(msg)
	}()

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Booking successful",
	})
}

// GetMyAppointments godoc
// @Summary      Get student's booked appointments
// @Description  Returns a list of all upcoming and past appointments booked by the logged-in student.
// @Tags         student-booking
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.StudentAppointmentResponse
// @Failure      403  {object}  models.ErrorResponse "Not authorized"
// @Failure      500  {object}  models.ErrorResponse "Database error"
// @Router       /student/appointments [get]
func (h *BookingHandler) GetMyAppointments(c *gin.Context) {
	studentID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "student" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Only students can view their appointments",
		})
		return
	}

	// Fetch slots from DB booked by this student
	var slots []models.Slot
	if err := config.DB.
		Where("student_id = ? AND is_booked = ?", studentID, true).
		Order("start_time asc").
		Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Database error",
		})
		return
	}

	if len(slots) == 0 {
		c.JSON(http.StatusOK, []models.StudentAppointmentResponse{})
		return
	}

	// Extract unique Psychologist IDs to fetch their names
	var psychIDs []string
	uniquePsychs := make(map[string]bool)
	for _, s := range slots {
		if !uniquePsychs[s.PsychologistID] {
			uniquePsychs[s.PsychologistID] = true
			psychIDs = append(psychIDs, s.PsychologistID)
		}
	}

	// Call User Service via gRPC
	psychMap := make(map[string]string)
	grpcResp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{
		Ids: psychIDs,
	})

	if err == nil {
		for _, p := range grpcResp.Profiles {
			psychMap[p.Id] = p.FullName
		}
	} else {
		log.Printf("Failed to fetch psychologist profiles: %v", err)
	}

	var response []models.StudentAppointmentResponse
	for _, s := range slots {
		psychName := "Unknown Specialist"
		if name, ok := psychMap[s.PsychologistID]; ok {
			psychName = name
		}

		response = append(response, models.StudentAppointmentResponse{
			ID:                   s.ID,
			StartTime:            s.StartTime,
			Duration:             s.Duration,
			BookingType:          s.BookingType,
			PsychologistID:       s.PsychologistID,
			PsychologistName:     psychName,
			QuestionnaireAnswers: s.QuestionnaireAnswers,
		})
	}

	c.JSON(http.StatusOK, response)
}

// CancelAppointment godoc
// @Summary      Cancel a booked appointment
// @Description  Student cancels their own booking. This frees up the slot for other students to book.
// @Tags         student-booking
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Slot ID (UUID)"
// @Success      200  {object}  models.MessageResponse
// @Failure      400  {object}  models.ErrorResponse "Invalid request"
// @Failure      403  {object}  models.ErrorResponse "Not authorized or not the owner"
// @Failure      404  {object}  models.ErrorResponse "Slot not found"
// @Failure      409  {object}  models.ErrorResponse "Slot is not booked"
// @Failure      500  {object}  models.ErrorResponse "Database error"
// @Router       /student/slots/{id}/cancel [post]
func (h *BookingHandler) CancelAppointment(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "student" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Only students can cancel their appointments",
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

	// Check if it's actually booked
	if !slot.IsBooked || slot.StudentID == nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "This slot is not currently booked",
		})
		return
	}

	// Security Check: Make sure the student owns this booking
	if *slot.StudentID != studentID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "You can only cancel your own appointments",
		})
		return
	}

	psychID := slot.PsychologistID

	result := config.DB.Model(&models.Slot{}).
		Where("id = ? AND version = ?", slot.ID, slot.Version).
		Updates(map[string]interface{}{
			"is_booked":             false,
			"student_id":            nil,
			"booking_type":          "",
			"questionnaire_answers": "",
			"version":               slot.Version + 1,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Database error",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Could not cancel. Please try again.",
		})
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
				Type:    "booking_cancellation",
				ToEmail: studentEmail,
				Data: map[string]string{
					"psychologist_name": psychName,
					"datetime":          slot.StartTime.Format("Monday, 02 Jan 2006 at 15:04"),
				},
			}
			h.RabbitMQ.PublishNotification(msg)
		}

		dateStr := slot.StartTime.Format("2006-01-02")
		h.notifyWaitlist(psychID, dateStr, psychName)
	}()

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Appointment successfully canceled",
	})
}

// RescheduleAppointment godoc
// @Summary      Reschedule an appointment
// @Description  Student moves their booking from one slot to another available slot. This is done atomically so they don't lose their original slot if the new one is taken.
// @Tags         student-booking
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                  true  "Old Slot ID (The one currently booked)"
// @Param        request  body      models.RescheduleInput  true  "The new Slot ID to move to"
// @Success      200      {object}  models.MessageResponse
// @Failure      400      {object}  models.ErrorResponse "Invalid request"
// @Failure      403      {object}  models.ErrorResponse "Not authorized or not the owner"
// @Failure      404      {object}  models.ErrorResponse "Slot not found"
// @Failure      409      {object}  models.ErrorResponse "New slot already booked or race condition"
// @Failure      500      {object}  models.ErrorResponse "Database error"
// @Router       /student/slots/{id}/reschedule [post]
func (h *BookingHandler) RescheduleAppointment(c *gin.Context) {
	oldSlotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "student" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only students can reschedule"})
		return
	}

	var input models.RescheduleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	if oldSlotID == input.NewSlotID {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Old slot and new slot cannot be the same"})
		return
	}

	// START TRANSACTION
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Fetch Old Slot
	var oldSlot models.Slot
	if err := tx.First(&oldSlot, "id = ?", oldSlotID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Old slot not found"})
		return
	}

	// Verify ownership
	if !oldSlot.IsBooked || oldSlot.StudentID == nil || *oldSlot.StudentID != studentID {
		tx.Rollback()
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "You can only reschedule your own active appointments"})
		return
	}

	// 2. Fetch New Slot
	var newSlot models.Slot
	if err := tx.First(&newSlot, "id = ?", input.NewSlotID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "New slot not found"})
		return
	}

	// Verify availability
	if newSlot.IsBooked {
		tx.Rollback()
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "The requested new slot is already booked by someone else"})
		return
	}

	// Ensure it's the same psychologist
	//if oldSlot.PsychologistID != newSlot.PsychologistID {
	//	tx.Rollback()
	//	c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Cannot reschedule to a different psychologist using this endpoint"})
	//	return
	//}

	// Free up the Old Slot
	res1 := tx.Model(&models.Slot{}).
		Where("id = ? AND version = ?", oldSlot.ID, oldSlot.Version).
		Updates(map[string]interface{}{
			"is_booked":             false,
			"student_id":            nil,
			"booking_type":          "",
			"questionnaire_answers": "",
			"version":               oldSlot.Version + 1,
		})

	if res1.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Could not modify the old appointment. Please try again."})
		return
	}

	// Book the New Slot (Transferring the data from the old one)
	res2 := tx.Model(&models.Slot{}).
		Where("id = ? AND version = ?", newSlot.ID, newSlot.Version).
		Updates(map[string]interface{}{
			"is_booked":             true,
			"student_id":            studentID,
			"booking_type":          oldSlot.BookingType,
			"questionnaire_answers": oldSlot.QuestionnaireAnswers,
			"version":               newSlot.Version + 1,
		})

	if res2.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "The new slot was just taken! Your original appointment was kept."})
		return
	}

	// COMMIT TRANSACTION
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Transaction failed"})
		return
	}

	go func() {
		resp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{studentID, newSlot.PsychologistID},
		})

		var studentEmail, psychName string
		if err == nil {
			for _, p := range resp.Profiles {
				if p.Id == studentID {
					studentEmail = p.Email
				} else if p.Id == newSlot.PsychologistID {
					psychName = p.FullName
				}
			}
		}

		if studentEmail != "" {
			msg := clients.NotificationMessage{
				Type:    "booking_reschedule",
				ToEmail: studentEmail,
				Data: map[string]string{
					"psychologist_name": psychName,
					"datetime":          newSlot.StartTime.Format("Monday, 02 Jan 2006 at 15:04"),
					"format":            oldSlot.BookingType,
				},
			}
			h.RabbitMQ.PublishNotification(msg)
		}

		oldDateStr := oldSlot.StartTime.Format("2006-01-02")
		h.notifyWaitlist(oldSlot.PsychologistID, oldDateStr, psychName)
	}()

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Appointment successfully rescheduled",
	})
}
