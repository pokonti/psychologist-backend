package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/clients"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/handlers"
)

func main() {
	r := gin.Default()
	config.ConnectDB()

	// Init gRPC Client
	userClient, conn, err := clients.NewUserProfileClient()
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer conn.Close()

	// Init Handler
	h := &handlers.BookingHandler{
		UserClient: userClient,
	}

	// Routes
	// The Gateway sends the full path "/api/v1/...", so we must listen for it.
	api := r.Group("/api/v1")
	{
		// Public / Shared Routes (Get Slots)
		// Gateway: GET /api/v1/slots
		api.GET("/slots", h.GetAvailableSlots)
		api.GET("/slots/calendar", h.GetCalendarAvailability)

		// Psychologist Routes
		// Gateway: POST /api/v1/psychologist/slots
		psych := api.Group("/psychologist")
		{
			psych.POST("/slots", handlers.CreateSlot)
		}

		// Student Routes
		// Gateway: POST /api/v1/student/slots/:id/book
		student := api.Group("/student")
		{
			student.POST("/slots/:id/book", handlers.BookSlot)
		}
	}

	log.Println("Booking Service running on port 8084")
	r.Run(":8084")
}
