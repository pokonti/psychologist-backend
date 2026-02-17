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
	studentID := c.GetHeader("X-User-ID")
	userRole := c.GetHeader("X-User-Role")
	slotID := c.Param("id")

	if userRole != "student" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied. Only students can book appointments.",
		})
		return
	}

	// START TRANSACTION (Prevent Double Booking)
	tx := config.DB.Begin()

	var slot models.Slot
	// Lock the row specifically for update so no one else can read/write it
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&slot, "id = ?", slotID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Slot not found"})
		return
	}

	if slot.IsBooked {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "Slot is already booked"})
		return
	}

	// Update Slot
	slot.IsBooked = true
	slot.StudentID = &studentID

	if err := tx.Save(&slot).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to book slot"})
		return
	}

	// Commit Transaction
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Booking successful", "slot": slot})
}
