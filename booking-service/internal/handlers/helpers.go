package handlers

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/clients"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

type BookingHandler struct {
	UserClient userprofile.UserProfileServiceClient
	RabbitMQ   *clients.RabbitMQClient
}

// Helper to parse "2026-01-01"
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func combineDateAndTime(date time.Time, timeStr string) (time.Time, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, err
	}

	loc := time.FixedZone("Asia/Almaty", 5*60*60)

	localTime := time.Date(
		date.Year(), date.Month(), date.Day(),
		t.Hour(), t.Minute(), 0, 0, loc,
	)

	return localTime.UTC(), nil
}

// Helper function to get the start and end of a week for a given date
// Assuming Monday is the first day of the week
func getWeekRange(date time.Time) (time.Time, time.Time) {
	// Find how many days we are past Monday (0 = Sunday, 1 = Monday, etc.)
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Make Sunday the last day of the week (7) instead of 0
	}

	// Subtract days to get to Monday 00:00:00
	startOfWeek := date.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)

	// Add 7 days to get to next Monday 00:00:00 (which is the exclusive end of this week)
	endOfWeek := startOfWeek.AddDate(0, 0, 7)

	return startOfWeek, endOfWeek
}

func logBookingAction(slotID, psychID, studentID, action, topic, message string) {
	logEntry := models.BookingLog{
		ID:             uuid.NewString(),
		SlotID:         slotID,
		PsychologistID: psychID,
		StudentID:      studentID,
		Action:         action,
		ReasonTopic:    topic,
		ReasonMessage:  message,
		Timestamp:      time.Now(),
	}

	go func() {
		if err := config.DB.Create(&logEntry).Error; err != nil {
			log.Printf("Failed to write booking log: %v", err)
		}
	}()
}
