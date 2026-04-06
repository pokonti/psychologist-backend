package worker

import (
	"context"
	"log"
	"time"

	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/clients"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

// StartReminderWorker checks for upcoming appointments and sends reminders
func StartReminderWorker(userClient userprofile.UserProfileServiceClient, rabbitMQ *clients.RabbitMQClient) {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			now := time.Now()

			// 24-Hour Reminders
			targetStart24 := now.Add(24 * time.Hour).Add(-5 * time.Minute)
			targetEnd24 := now.Add(24 * time.Hour)

			var slots24 []models.Slot
			config.DB.Where("status = ? AND start_time >= ? AND start_time <= ?", models.StatusBooked, targetStart24, targetEnd24).Find(&slots24)

			for _, slot := range slots24 {
				sendReminder(slot, userClient, rabbitMQ, "tomorrow")
			}

			// 2-Hour Reminders
			//targetStart2 := now.Add(2 * time.Hour).Add(-5 * time.Minute)
			//targetEnd2 := now.Add(2 * time.Hour)
			targetStart1 := now
			targetEnd1 := now.Add(10 * time.Minute)

			var slots1 []models.Slot
			config.DB.Where("status = ? AND start_time >= ? AND start_time <= ?", models.StatusBooked, targetStart1, targetEnd1).Find(&slots1)

			for _, slot := range slots1 {
				sendReminder(slot, userClient, rabbitMQ, "in 2 hours")
			}
		}
	}()
}

func sendReminder(slot models.Slot, userClient userprofile.UserProfileServiceClient, rabbitMQ *clients.RabbitMQClient, subject string) {
	if slot.StudentID == nil {
		return
	}

	resp, err := userClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
		Ids: []string{*slot.StudentID, slot.PsychologistID},
	})

	var studentEmail, psychName, telegramChatID string

	if err == nil {
		for _, p := range resp.Profiles {
			if p.Id == *slot.StudentID {
				studentEmail = p.Email
				telegramChatID = p.TelegramChatId
			} else if p.Id == slot.PsychologistID {
				psychName = p.FullName
			}
		}
	}

	if studentEmail != "" {
		msg := clients.NotificationMessage{
			Type:    "session_reminder",
			ToEmail: studentEmail,
			Data: map[string]string{
				"psychologist_name": psychName,
				"datetime":          slot.StartTime.Format("Monday, 02 Jan 2006 at 15:04"),
				"telegram_chat_id":  telegramChatID,
				"subject":           subject,
			},
		}
		rabbitMQ.PublishNotification(msg)
		log.Printf("[Worker] Sent reminder for slot %s", slot.ID)
	}
}
