package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/internal/handlers"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(r *gin.Engine, h *handlers.BookingHandler) {

	api := r.Group("/api/v1")
	{
		// Shared routes
		api.GET("/slots", h.GetAvailableSlots)
		api.GET("/slots/calendar", h.GetCalendarAvailability)

		// Psychologist routes
		psych := api.Group("/psychologist")
		{
			psych.POST("/slots", handlers.CreateSlot)
		}

		// Student routes
		student := api.Group("/student")
		{
			student.POST("/slots/:id/book", handlers.BookSlot)
		}
	}

	// Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
