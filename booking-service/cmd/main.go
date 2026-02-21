package cmd

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
	r.POST("/slots", handlers.CreateSlot) // Psych only
	r.GET("/slots", h.GetAvailableSlots)
	r.GET("/slots/calendar", h.GetCalendarAvailability) // Get valid days for month
	r.POST("/slots/:id/book", handlers.BookSlot)        // Student only

	log.Println("Booking Service running on port 8084")
	r.Run(":8084")
}
