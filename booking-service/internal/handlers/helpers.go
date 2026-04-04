package handlers

import (
	"context"
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

// Helper to merge date "2026-01-01" with time "14:30"
func combineDateAndTime(date time.Time, timeStr string) (time.Time, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, err
	}
	// Return new date with specific hour/minute
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC), nil
}

// notifyWaitlist is a background helper that finds waitlisted students and emails them
func (h *BookingHandler) notifyWaitlist(psychID string, dateStr string, psychName string) {
	var waitlist []models.WaitlistEntry
	config.DB.Where("psychologist_id = ? AND date = ?", psychID, dateStr).Find(&waitlist)

	if len(waitlist) == 0 {
		return
	}

	var studentIDs []string
	for _, w := range waitlist {
		studentIDs = append(studentIDs, w.StudentID)
	}

	resp, err := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
		Ids: studentIDs,
	})
	if err != nil {
		log.Printf("Failed to fetch waitlist users: %v", err)
		return
	}

	for _, profile := range resp.Profiles {
		if profile.Email != "" {
			msg := clients.NotificationMessage{
				Type:    "waitlist_alert",
				ToEmail: profile.Email,
				Data: map[string]string{
					"psychologist_name": psychName,
					"date":              dateStr,
				},
			}
			h.RabbitMQ.PublishNotification(msg)
		}
	}
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
