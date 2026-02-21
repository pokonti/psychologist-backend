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

// Helper to parse "2026-01-01"
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// Helper to merge date "2026-01-01" with time "14:30"
func combineDateAndTime(date time.Time, timeStr string) (time.Time, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, err
	}
	// Return new date with specific hour/minute
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC), nil
}

type BookingHandler struct {
	UserClient userprofile.UserProfileServiceClient
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date"})
		return
	}
	endDate, err := parseDate(input.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "No slots created. Check your dates and schedule."})
		return
	}

	// 4. Batch Insert
	if err := config.DB.Create(&slotsToCreate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create slots"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Schedule created successfully",
		"count":   len(slotsToCreate),
	})
}

// GET /slots/calendar?psychologist_id=...&year=2026&month=2
func (h *BookingHandler) GetCalendarAvailability(c *gin.Context) {
	psychID := c.Query("psychologist_id")
	year := c.Query("year")
	month := c.Query("month")

	if psychID == "" || year == "" || month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing params"})
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

	c.JSON(http.StatusOK, gin.H{"available_dates": availableDays})
}

// GET /slots?psychologist_id=...&date=2026-02-20
func (h *BookingHandler) GetAvailableSlots(c *gin.Context) {
	psychID := c.Query("psychologist_id")
	dateStr := c.Query("date")

	if psychID == "" || dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing params"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}

	// --- gRPC Enrichment ---

	// Collect ID
	psychIDList := []string{psychID} // In this case, we only have one ID because of the filter

	// Call User Service
	grpcResp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{
		Ids: psychIDList,
	})

	psychName := "Unknown Specialist"
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
			PsychologistName: psychName, // <--- Data from gRPC
		})
	}

	c.JSON(http.StatusOK, response)
}

// BookSlot Student books a slot
func BookSlot(c *gin.Context) {
	slotID := c.Param("id")
	studentID := c.GetHeader("X-User-ID")

	// 1. Read the slot (Standard read, NO LOCKS)
	var slot models.Slot
	if err := config.DB.First(&slot, "id = ?", slotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slot not found"})
		return
	}

	// 2. Early Check
	if slot.IsBooked {
		c.JSON(http.StatusConflict, gin.H{"error": "Slot is already booked"})
		return
	}

	// 3. OPTIMISTIC UPDATE
	// SQL Generated:
	// UPDATE slots SET is_booked=true, student_id='...', version=2
	// WHERE id='...' AND version=1

	// We increment version manually to invalidate other requests
	result := config.DB.Model(&models.Slot{}).
		Where("id = ? AND version = ?", slot.ID, slot.Version).
		Updates(map[string]interface{}{
			"is_booked":  true,
			"student_id": studentID,
			"version":    slot.Version + 1, // Increment Version
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 4. Check if the row was actually touched
	if result.RowsAffected == 0 {
		// If 0 rows were updated, it means the Version changed between
		// step 1 and step 3 (Someone else booked it milliseconds ago)
		c.JSON(http.StatusConflict, gin.H{"error": "Slot was just booked by someone else"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Booking successful"})
}
