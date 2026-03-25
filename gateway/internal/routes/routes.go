package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/gateway/internal/middleware"
	"github.com/pokonti/psychologist-backend/gateway/internal/proxy"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	api.Use(middleware.SetupRateLimiter())

	authGroup := api.Group("/auth", middleware.SetupAuthLimiter())
	{
		authGroup.POST("/register", proxy.Forward("http://auth-service:8083"))
		authGroup.POST("/login", proxy.Forward("http://auth-service:8083"))
		authGroup.POST("/verify", proxy.Forward("http://auth-service:8083"))
		authGroup.POST("/refresh", proxy.Forward("http://auth-service:8083"))
	}

	// Protected routes
	protected := api.Group("", middleware.JWTAuth())
	protected.GET("/users/me", proxy.Forward("http://user-service:8081"))
	protected.PUT("/users/me", proxy.Forward("http://user-service:8081"))
	protected.GET("/users/psychologists", proxy.Forward("http://user-service:8081"))
	protected.POST("/users/me/mood", proxy.Forward("http://user-service:8081"))
	protected.GET("/users/me/mood/graphic", proxy.Forward("http://user-service:8081"))
	protected.GET("/slots", proxy.Forward("http://booking-service:8084"))
	protected.GET("/slots/calendar", proxy.Forward("http://booking-service:8084"))
	protected.POST("/auth/logout", proxy.Forward("http://auth-service:8083"))

	// Psychologist
	psychOnly := protected.Group("/psychologist", middleware.RequireRoles("psychologist", "admin"))
	{
		psychOnly.POST("/slots", proxy.Forward("http://booking-service:8084"))
		psychOnly.GET("/slots", proxy.Forward("http://booking-service:8084"))
		psychOnly.DELETE("/slots/:id", proxy.Forward("http://booking-service:8084"))
		psychOnly.PUT("/slots/:id/notes", proxy.Forward("http://booking-service:8084"))
		psychOnly.GET("/students/:student_id/history", proxy.Forward("http://booking-service:8084"))
		psychOnly.POST("/slots/:id/cancel", proxy.Forward("http://booking-service:8084"))
		psychOnly.PUT("/slots/:id/recommendations", proxy.Forward("http://booking-service:8084"))
		psychOnly.GET("/reviews", proxy.Forward("http://booking-service:8084"))
		psychOnly.GET("/statistics", proxy.Forward("http://booking-service:8084"))
	}

	// Student
	studentOnly := protected.Group("/student", middleware.RequireRoles("student", "admin"))
	{
		studentOnly.POST("/slots/:id/reserve", proxy.Forward("http://booking-service:8084"))
		studentOnly.POST("/slots/:id/confirm", proxy.Forward("http://booking-service:8084"))
		studentOnly.GET("/appointments", proxy.Forward("http://booking-service:8084"))
		studentOnly.POST("/slots/:id/cancel", proxy.Forward("http://booking-service:8084"))
		studentOnly.POST("/slots/:id/reschedule", proxy.Forward("http://booking-service:8084"))
		studentOnly.POST("/waitlist", proxy.Forward("http://booking-service:8084"))
		studentOnly.GET("/waitlist", proxy.Forward("http://booking-service:8084"))
		studentOnly.DELETE("/waitlist/:id", proxy.Forward("http://booking-service:8084"))
		studentOnly.POST("/slots/:id/rate", proxy.Forward("http://booking-service:8084"))

	}

	adminOnly := protected.Group("/admin", middleware.RequireRoles("admin"))
	{
		adminOnly.GET("/dashboard", proxy.Forward("http://booking-service:8084"))
		adminOnly.GET("/bookings", proxy.Forward("http://booking-service:8084"))
		adminOnly.POST("/bookings/:id/cancel", proxy.Forward("http://booking-service:8084"))
		adminOnly.POST("/users", proxy.Forward("http://auth-service:8083"))
		adminOnly.PATCH("/users/:id/block", proxy.Forward("http://auth-service:8083"))
		adminOnly.GET("/users", proxy.Forward("http://user-service:8081"))
		adminOnly.GET("/psychologists", proxy.Forward("http://user-service:8081"))
		adminOnly.GET("/reviews", proxy.Forward("http://booking-service:8084"))
	}

	// Proxy Swagger UIs
	// Access Auth docs at: http://localhost:8080/docs/auth/index.html
	r.Any("/docs/auth/*any", proxy.Forward("http://auth-service:8083/swagger"))

	// Access Booking docs at: http://localhost:8080/docs/booking/index.html
	r.Any("/docs/booking/*any", proxy.Forward("http://booking-service:8084/swagger"))

	// Access User docs at: http://localhost:8080/docs/user/index.html
	r.Any("/docs/user/*any", proxy.Forward("http://user-service:8081/swagger"))

}
