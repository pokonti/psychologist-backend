package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
)

type CreateSlotInput struct {
	StartTime time.Time `json:"start_time" binding:"required"`
}

// CreateSlot Psychologist creates a slot
func CreateSlot(c *gin.Context) {
	// Get User ID from Header
	psychologistID := c.GetHeader("X-User-ID")
	role := c.GetHeader("X-User-Role")

	if role != "psychologist" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only psychologists can create slots"})
		return
	}

	var input CreateSlotInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slot := models.Slot{
		ID:             uuid.NewString(),
		PsychologistID: psychologistID,
		StartTime:      input.StartTime,
		IsBooked:       false,
	}

	if err := config.DB.Create(&slot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create slot"})
		return
	}

	c.JSON(http.StatusCreated, slot)
}

// GetAvailableSlots shows available slots
func GetAvailableSlots(c *gin.Context) {
	var slots []models.Slot
	if err := config.DB.Where("is_booked = ?", false).Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch slots"})
		return
	}
	c.JSON(http.StatusOK, slots)
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
