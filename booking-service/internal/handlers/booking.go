package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

type BookingHandler struct {
	UserClient userprofile.UserProfileServiceClient
}

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

	// 4. Batch Insert
	if err := config.DB.Create(&slotsToCreate).Error; err != nil {
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
// @Param        request    body    models.BookSlotInput  true  "Booking type (online/offline)"
// @Success 200 {object} models.MessageResponse
// @Failure      400  {object}  models.ErrorResponse "invalid request body"
// @Failure      404  {object}  models.ErrorResponse "slot not found"
// @Failure      409  {object}  models.ErrorResponse "slot already booked or just booked"
// @Failure      500  {object}  models.ErrorResponse "database error"
// @Router       /student/slots/{id}/book [post]
func BookSlot(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")

	var input models.BookSlotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			"is_booked":    true,
			"student_id":   studentID,
			"booking_type": input.BookingType,
			"version":      slot.Version + 1, // Increment Version
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
		// step 1 and step 3 (Someone else booked it milliseconds ago)
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "Slot was just booked by someone else",
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Booking successful",
	})
}
