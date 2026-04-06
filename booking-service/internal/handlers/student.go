package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/clients"
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
		Where("status = ?", models.StatusAvailable).
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
			Status:           s.Status,
			PsychologistID:   s.PsychologistID,
			PsychologistName: psychName,
			MeetingLink:      s.MeetingLink,
		})
	}

	c.JSON(http.StatusOK, response)
}

// ReserveSlot godoc
// @Summary      Book a slot as a student
// @Description  Student books a specific slot. Uses optimistic locking on the version field to avoid double-booking.
// @Tags         student-booking
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Param        id         path    string         true  "Slot ID"
// @Param        request  body   models.ReserveSlotInput  true  "Booking details: type"
// @Success 200 {object} models.MessageResponse
// @Failure      400  {object}  models.ErrorResponse "invalid request body"
// @Failure      404  {object}  models.ErrorResponse "slot not found"
// @Failure      409  {object}  models.ErrorResponse "slot already booked or just booked"
// @Failure      500  {object}  models.ErrorResponse "database error"
// @Router       /student/slots/{id}/book [post]
// 1. RESERVE SLOT (Step 1 of booking)
// @Router /student/slots/{id}/reserve [post]
func (h *BookingHandler) ReserveSlot(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")

	var input models.ReserveSlotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Slot not found"})
		return
	}

	if slot.Status != models.StatusAvailable {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Slot is no longer available"})
		return
	}

	// Calculate boundaries for the requested slot's date
	startOfDay := slot.StartTime.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)
	startOfWeek, endOfWeek := getWeekRange(slot.StartTime)

	var dailyCount, weeklyCount int64

	// Check Daily Limit (Max 1 per day)
	config.DB.Model(&models.Slot{}).
		Where("student_id = ? AND status IN ?", studentID, []string{models.StatusReserved, models.StatusBooked}).
		Where("start_time >= ? AND start_time < ?", startOfDay, endOfDay).
		Count(&dailyCount)

	if dailyCount >= 1 {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "You can only book 1 appointment per day.",
		})
		return
	}

	// Check Weekly Limit (Max 2 per week)
	config.DB.Model(&models.Slot{}).
		Where("student_id = ? AND status IN ?", studentID, []string{models.StatusReserved, models.StatusBooked}).
		Where("start_time >= ? AND start_time < ?", startOfWeek, endOfWeek).
		Count(&weeklyCount)

	if weeklyCount >= 2 {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "You have reached the maximum limit of 2 appointments per week.",
		})
		return
	}

	now := time.Now()

	// Optimistic Update to Reserved
	res := config.DB.Model(&models.Slot{}).
		Where("id = ? AND version = ?", slot.ID, slot.Version).
		Updates(map[string]interface{}{
			"status":       models.StatusReserved,
			"student_id":   studentID,
			"booking_type": input.BookingType,
			"reserved_at":  now,
			"version":      slot.Version + 1,
		})

	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Slot was just taken by someone else"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Slot reserved for 20 minutes. Please complete the questionnaire.",
		"expires_at": now.Add(20 * time.Minute),
	})
}

// ConfirmSlot godoc
// @Summary      Confirm a booked appointment
// @Description  Finalizes the reservation by submitting the questionnaire and student's phone number. Requires a previous 'reserve' action.
// @Tags         student-booking
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path   string               true  "Slot ID (Must be currently 'reserved')"
// @Param        request body   models.ConfirmSlotInput true  "Booking details (phone, answers)"
// @Success      200 {object}   models.MessageResponse
// @Failure      400 {object}   models.ErrorResponse "Invalid request body"
// @Failure      403 {object}   models.ErrorResponse "Not authorized or no active reservation"
// @Failure      404 {object}   models.ErrorResponse "Slot not found"
// @Failure      409 {object}   models.ErrorResponse "Reservation expired or conflict"
// @Failure      500 {object}   models.ErrorResponse "Database or gRPC error"
// @Router       /student/slots/{id}/confirm [post]
func (h *BookingHandler) ConfirmSlot(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")

	var input models.ConfirmSlotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error()})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Slot not found"})
		return
	}

	if slot.Status != models.StatusReserved || slot.StudentID == nil || *slot.StudentID != studentID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "You do not have an active reservation for this slot"})
		return
	}

	if slot.ReservedAt != nil && time.Since(*slot.ReservedAt) > 15*time.Minute {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Your reservation has expired"})
		return
	}

	res := config.DB.Model(&models.Slot{}).
		Where("id = ? AND version = ?", slot.ID, slot.Version).
		Updates(map[string]interface{}{
			"status":                models.StatusBooked,
			"questionnaire_answers": input.Answers,
			"phone_number":          input.PhoneNumber,
			"version":               slot.Version + 1,
		})

	if res.RowsAffected == 0 {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to confirm booking"})
		return
	}

	go func() {
		grpcResp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{studentID, slot.PsychologistID},
		})

		var studentEmail, psychName, tgID string
		if err == nil {
			for _, p := range grpcResp.Profiles {
				if p.Id == studentID {
					studentEmail = p.Email
					tgID = p.TelegramChatId
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
				"format":            slot.BookingType,
				"start_time_raw":    slot.StartTime.Format(time.RFC3339),
				"slot_id":           slot.ID,
				"telegram_chat_id":  tgID,
			},
		}

		h.RabbitMQ.PublishNotification(msg)

		_, err = h.UserClient.UpdateUserPhone(context.Background(), &userprofile.UpdateUserPhoneRequest{
			Id:    studentID,
			Phone: input.PhoneNumber,
		})
		if err != nil {
			log.Printf("Failed to sync phone to user-service: %v", err)
		}
	}()

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Appointment confirmed successfully!"})
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
		Where("student_id = ? AND status = ?", studentID, models.StatusBooked).
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
			ID:                     s.ID,
			StartTime:              s.StartTime,
			Duration:               s.Duration,
			BookingType:            s.BookingType,
			PsychologistID:         s.PsychologistID,
			PsychologistName:       psychName,
			QuestionnaireAnswers:   s.QuestionnaireAnswers,
			StudentRecommendations: s.StudentRecommendations,
			MeetingLink:            s.MeetingLink,
		})
	}

	c.JSON(http.StatusOK, response)
}

// CancelAppointment godoc
// @Summary      Cancel a booked appointment
// @Description  Student cancels their own booking. This resets the slot to 'available' and notifies the waitlist.
// @Tags         student-booking
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Slot ID (UUID)"
// @Param        request body models.CancellationInput true "Cancellation details"
// @Success      200  {object}  models.MessageResponse "Appointment successfully canceled"
// @Failure      400  {object}  models.ErrorResponse   "Invalid request"
// @Failure      403  {object}  models.ErrorResponse   "Not authorized or not the owner"
// @Failure      404  {object}  models.ErrorResponse   "Slot not found"
// @Failure      409  {object}  models.ErrorResponse   "Slot is not booked"
// @Failure      500  {object}  models.ErrorResponse   "Database error"
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

	var input models.CancellationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Cancellation reason is required"})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Slot not found",
		})
		return
	}

	if slot.Status != models.StatusBooked || slot.StudentID == nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "This slot is not currently booked",
		})
		return
	}

	// Security Check
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
			"status":                models.StatusAvailable,
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

	logBookingAction(slot.ID, slot.PsychologistID, *slot.StudentID, "canceled_by_student", input.ReasonTopic, input.ReasonMessage)

	go func() {
		resp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{studentID, psychID},
		})

		var studentEmail, psychName, tgID string
		if err == nil {
			for _, p := range resp.Profiles {
				if p.Id == studentID {
					studentEmail = p.Email
					tgID = p.TelegramChatId
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
					"telegram_chat_id":  tgID,
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
// @Description  Student moves their booking from one slot to another available slot atomically. Alerts waitlist for the old slot.
// @Tags         student-booking
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                  true  "Old Slot ID (The one currently booked)"
// @Param        request  body      models.RescheduleInput  true  "The new Slot ID to move to"
// @Success      200      {object}  models.MessageResponse  "Appointment successfully rescheduled"
// @Failure      400      {object}  models.ErrorResponse    "Invalid request body or same slot IDs"
// @Failure      403      {object}  models.ErrorResponse    "Not authorized or not the owner"
// @Failure      404      {object}  models.ErrorResponse    "Slot not found"
// @Failure      409      {object}  models.ErrorResponse    "New slot already booked or race condition"
// @Failure      500      {object}  models.ErrorResponse    "Database error"
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

	if oldSlot.Status != models.StatusBooked || oldSlot.StudentID == nil || *oldSlot.StudentID != studentID {
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

	if newSlot.Status != models.StatusAvailable {
		tx.Rollback()
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "The requested new slot is no longer available"})
		return
	}

	// Free up the Old Slot
	res1 := tx.Model(&models.Slot{}).
		Where("id = ? AND version = ?", oldSlot.ID, oldSlot.Version).
		Updates(map[string]interface{}{
			"status":                models.StatusAvailable,
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
			"status":                models.StatusBooked,
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

// RateSession godoc
// @Summary      Rate a completed session
// @Description  Student leaves a 1-5 star rating and an optional review for a completed appointment.
// @Tags         student-booking
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path   string            true  "Slot ID"
// @Param        request body   models.RateSessionInput  true  "Rating and Review"
// @Success      200 {object} models.MessageResponse
// @Failure      400 {object} models.ErrorResponse "Invalid rating (must be 1-5) or session not finished"
// @Failure      403 {object} models.ErrorResponse "Not authorized"
// @Failure      404 {object} models.ErrorResponse "Slot not found"
// @Failure      409 {object} models.ErrorResponse "Session already rated"
// @Router       /student/slots/{id}/rate [post]
func (h *BookingHandler) RateSession(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "student" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Only students can rate sessions"})
		return
	}

	var input models.RateSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Slot not found"})
		return
	}

	if slot.Status != models.StatusBooked || slot.StudentID == nil || *slot.StudentID != studentID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "You can only rate your own booked sessions"})
		return
	}

	// Time Check
	if time.Now().Before(slot.StartTime) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "You cannot rate a session that hasn't happened yet"})
		return
	}

	// Duplicate Check: Ensure they haven't rated it already
	if slot.Rating > 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "You have already rated this session"})
		return
	}

	// Save the Rating
	result := config.DB.Model(&models.Slot{}).Where("id = ?", slotID).Updates(map[string]interface{}{
		"rating": input.Rating,
		"review": input.Review,
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error while saving rating"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Could not save rating. Please try again."})
		return
	}

	go func() {
		msg := clients.UserEventMessage{
			Type:           "new_rating",
			PsychologistID: slot.PsychologistID,
			Rating:         input.Rating,
		}
		h.RabbitMQ.PublishUserEvent(msg)
	}()

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Thank you for your feedback!"})
}

func (h *BookingHandler) InternalUpdateMeetingLink(c *gin.Context) {
	slotID := c.Param("id")
	var input models.UpdateLinkInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := config.DB.Model(&models.Slot{}).Where("id = ?", slotID).Update("meeting_link", input.MeetingLink).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Link updated"})
}
