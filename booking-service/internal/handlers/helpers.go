package handlers

import (
	"context"
	"log"
	"time"

	"github.com/pokonti/psychologist-backend/booking-service/clients"
	"github.com/pokonti/psychologist-backend/booking-service/config"
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
