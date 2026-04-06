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
			psych.GET("/slots", h.GetMySchedule)
			psych.DELETE("/slots/:id", h.DeleteSlot)
			psych.PUT("/slots/:id/notes", h.AddSessionNote)
			psych.GET("/students/:student_id/history", h.GetStudentHistory)
			psych.POST("/slots/:id/cancel", h.CancelBookingByPsychologist)
			psych.PUT("/slots/:id/recommendations", h.AddRecommendation)
			psych.GET("/reviews", h.GetMyReviews)
			psych.GET("/statistics", h.GetPsychologistStats)
		}

		// Student routes
		student := api.Group("/student")
		{
			student.POST("/slots/:id/reserve", h.ReserveSlot)
			student.POST("/slots/:id/confirm", h.ConfirmSlot)
			student.GET("/appointments", h.GetMyAppointments)
			student.POST("/slots/:id/cancel", h.CancelAppointment)
			student.POST("/slots/:id/reschedule", h.RescheduleAppointment)

			student.POST("/waitlist", h.JoinWaitlist)
			student.GET("/waitlist", h.GetMyWaitlist)
			student.DELETE("/waitlist/:id", h.LeaveWaitlist)
			student.POST("/slots/:id/rate", h.RateSession)
		}
	}

	admin := api.Group("/admin")
	{
		admin.GET("/bookings", h.GetAllBookings)
		admin.POST("/bookings/:id/cancel", h.ForceCancelBooking)
		admin.GET("/admin/dashboard", h.GetDashboard)
		admin.GET("/reviews", h.GetAllReviews)
	}

	// Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	api.PUT("/internal/slots/:id/link", h.InternalUpdateMeetingLink)
}
