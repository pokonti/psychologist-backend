package consumer

import (
	"encoding/json"
	"log"

	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

type UserEventMessage struct {
	Type           string `json:"type"`
	PsychologistID string `json:"psychologist_id"`
	Rating         int    `json:"rating"`
}

func StartListening(ch *amqp.Channel, q amqp.Queue) {
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("[*] User Service is listening for user events...")

	var forever chan struct{}
	go func() {
		for d := range msgs {
			processMessage(d.Body)
		}
	}()
	<-forever
}

func processMessage(body []byte) {
	var msg UserEventMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	if msg.Type == "new_rating" {
		var user models.UserProfile
		if err := config.DB.First(&user, "id = ?", msg.PsychologistID).Error; err != nil {
			log.Printf("Psychologist not found: %s", msg.PsychologistID)
			return
		}

		// Calculate new average rating
		// Formula: ((Old Rating * Old Count) + New Rating) / (New Count)
		newCount := user.RatingCount + 1
		newRating := ((user.Rating * float32(user.RatingCount)) + float32(msg.Rating)) / float32(newCount)

		// Save to DB
		err := config.DB.Model(&user).Updates(map[string]interface{}{
			"rating":       newRating,
			"rating_count": newCount,
		}).Error

		if err != nil {
			log.Printf("Failed to update rating for %s: %v", msg.PsychologistID, err)
		} else {
			log.Printf("Updated psychologist %s to new rating: %.2f", msg.PsychologistID, newRating)
		}
	}
}
