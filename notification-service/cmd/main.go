package main

import (
	"log"

	"github.com/pokonti/psychologist-backend/notification-service/config"
	"github.com/pokonti/psychologist-backend/notification-service/internal/consumer"
)

func main() {
	log.Println("Starting Notification Service...")

	conn, ch, q := config.ConnectRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	consumer.StartListening(ch, q)
}
