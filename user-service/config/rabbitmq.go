package config

import (
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConnectRabbitMQ() (*amqp.Connection, *amqp.Channel, amqp.Queue) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	q, err := ch.QueueDeclare(
		"user_events_queue",
		true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	log.Println("User Service connected to RabbitMQ (Consumer)")
	return conn, ch, q
}
