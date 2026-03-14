package clients

import (
	"encoding/json"
	"log"

	"github.com/pokonti/psychologist-backend/auth-service/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct{}

// Message payload structure
type NotificationMessage struct {
	Type    string            `json:"type"`
	ToEmail string            `json:"to_email"`
	Data    map[string]string `json:"data"`
}

func NewRabbitMQClient() *RabbitMQClient {
	return &RabbitMQClient{}
}

func (r *RabbitMQClient) PublishNotification(msg NotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = config.RabbitChannel.Publish(
		"",
		config.RabbitQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	log.Printf("Message published to RabbitMQ: %s", msg.Type)
	return nil
}
