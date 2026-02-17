package cmd

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/handlers"
)

func main() {
	r := gin.Default()
	config.ConnectDB()

	// Routes
	r.POST("/slots", handlers.CreateSlot)        // Psych only
	r.GET("/slots", handlers.GetAvailableSlots)  // Everyone
	r.POST("/slots/:id/book", handlers.BookSlot) // Student only

	log.Println("Booking Service running on port 8084")
	r.Run(":8084")
}
