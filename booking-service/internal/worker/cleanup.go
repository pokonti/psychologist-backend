package worker

import (
	"log"
	"time"

	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
)

// StartReservationCleanup runs a background cron job to release expired locks
func StartReservationCleanup() {
	// Run the check every 1 minute
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			// Calculate the cutoff time (20 minutes ago)
			expiredTime := time.Now().Add(-20 * time.Minute)

			// Find and Update all expired reservations
			result := config.DB.Model(&models.Slot{}).
				Where("status = ? AND reserved_at < ?", models.StatusReserved, expiredTime).
				Updates(map[string]interface{}{
					"status":                models.StatusAvailable,
					"student_id":            nil,
					"reserved_at":           nil,
					"booking_type":          "",
					"questionnaire_answers": "",
				})

			if result.Error != nil {
				log.Printf("[Worker Error] Failed to clean up reservations: %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("[Worker] Successfully released %d expired reservations back to available", result.RowsAffected)
			}
		}
	}()
}
