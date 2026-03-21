package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/clients"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	_ "github.com/pokonti/psychologist-backend/booking-service/docs"
	"github.com/pokonti/psychologist-backend/booking-service/internal/handlers"
	"github.com/pokonti/psychologist-backend/booking-service/internal/worker"
	"github.com/pokonti/psychologist-backend/booking-service/routes"
)

// @title       KBTU Psychologist Booking Service API
// @version     1.0
// @description Slot scheduling and booking system for KBTU counseling platform.
// @BasePath    /api/v1
// @host        localhost:8080
// @schemes     http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	config.ConnectDB()
	config.ConnectRabbitMQ()

	defer config.RabbitConn.Close()
	defer config.RabbitChannel.Close()

	worker.StartReservationCleanup()

	// Init gRPC Client
	userClient, conn, err := clients.NewUserProfileClient()
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer conn.Close()

	rabbitMQ := clients.NewRabbitMQClient()
	// Init Handler
	h := &handlers.BookingHandler{
		UserClient: userClient,
		RabbitMQ:   rabbitMQ,
	}

	r := gin.Default()

	// Setup routes
	routes.SetupRoutes(r, h)

	log.Println("Booking Service running on port 8084")
	r.Run(":8084")
}
