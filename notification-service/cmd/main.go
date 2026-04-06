package main

import (
	"log"

	"github.com/pokonti/psychologist-backend/notification-service/config"
	"github.com/pokonti/psychologist-backend/notification-service/internal/clients"
	"github.com/pokonti/psychologist-backend/notification-service/internal/consumer"
)

func main() {
	log.Println("Starting Notification Service...")

	conn, ch, q := config.ConnectRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	bookingClient, conngrpc, _ := clients.NewBookingClient()
	defer conngrpc.Close()

	consumer.StartListening(ch, q, bookingClient)
}
