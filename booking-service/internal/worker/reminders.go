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

func StartReminderWorker(userClient userprofile.UserProfileServiceClient, rabbitMQ *clients.RabbitMQClient) {
	ticker := time.NewTicker(30 * time.Second)

	go func() {
		for range ticker.C {
			now := time.Now().UTC()

			targetStart := now.Add(-10 * time.Minute)
			targetEnd := now.Add(10 * time.Minute)

			var slots []models.Slot
			err := config.DB.Where("status = ? AND start_time >= ? AND start_time <= ?",
				models.StatusBooked, targetStart, targetEnd).Find(&slots).Error

			if err != nil {
				log.Printf("[Worker] DB Error: %v", err)
				continue
			}

			if len(slots) > 0 {
				log.Printf("[Worker] Found %d slots for reminders in current window", len(slots))
				for _, slot := range slots {
					sendReminder(slot, userClient, rabbitMQ, "upcoming")
				}
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
	} else {
		log.Printf("[Worker] gRPC Error fetching profiles: %v", err)
	}

	log.Printf("[Worker] Preparing reminder for %s. TG ID: %s", studentEmail, telegramChatID)

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
		log.Printf("[Worker] Published reminder to RabbitMQ for slot %s", slot.ID)
	}
}
