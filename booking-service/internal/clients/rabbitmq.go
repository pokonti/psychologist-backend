package clients

import (
	"encoding/json"
	"log"

	"github.com/pokonti/psychologist-backend/booking-service/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	q    amqp.Queue
}

type NotificationMessage struct {
	Type    string            `json:"type"`
	ToEmail string            `json:"to_email"`
	Data    map[string]string `json:"data"`
}

type UserEventMessage struct {
	Type           string `json:"type"`
	PsychologistID string `json:"psychologist_id"`
	Rating         int    `json:"rating"`
}

// NewRabbitMQClient creates a new publisher using the global config connection
func NewRabbitMQClient() *RabbitMQClient {
	return &RabbitMQClient{
		conn: config.RabbitConn,
		ch:   config.RabbitChannel,
		q:    config.RabbitQueue,
	}
}

func (r *RabbitMQClient) PublishNotification(msg NotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = r.ch.Publish(
		"",       // exchange
		r.q.Name, // routing key
		false,    // mandatory
		false,    // immediate
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

func (r *RabbitMQClient) PublishUserEvent(msg UserEventMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = r.ch.Publish(
		"", // exchange
		config.UserEventsQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		log.Printf("Failed to publish user event: %v", err)
		return err
	}
	return nil
}

// Close cleanly shuts down the connection
func (r *RabbitMQClient) Close() {
	r.ch.Close()
	r.conn.Close()
}
