package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/gateway/internal/middleware"
	"github.com/pokonti/psychologist-backend/gateway/internal/proxy"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	api.POST("/auth/register", proxy.Forward("http://auth-service:8083"))
	api.POST("/auth/login", proxy.Forward("http://auth-service:8083"))
	api.POST("/auth/verify", proxy.Forward("http://auth-service:8083"))

	// Protected routes
	protected := api.Group("", middleware.JWTAuth())
	protected.GET("/users/me", proxy.Forward("http://user-service:8081"))
	protected.PUT("/users/me", proxy.Forward("http://user-service:8081"))
	protected.GET("/users/psychologists", proxy.Forward("http://user-service:8081"))

	protected.GET("/slots", proxy.Forward("http://booking-service:8084"))
	protected.GET("/slots/calendar", proxy.Forward("http://booking-service:8084"))

	// Psychologist Only
	psychOnly := protected.Group("/psychologist", middleware.RequireRoles("psychologist", "admin"))
	{
		// This sends: /api/v1/psychologist/slots
		psychOnly.POST("/slots", proxy.Forward("http://booking-service:8084"))
	}

	// 3. Student Only
	studentOnly := protected.Group("/student", middleware.RequireRoles("student", "admin"))
	{
		// Book an appointment
		studentOnly.POST("/slots/:id/book", proxy.Forward("http://booking-service:8084"))
	}

	// Proxy Swagger UIs
	// Access Auth docs at: http://localhost:8080/docs/auth/index.html
	r.Any("/docs/auth/*any", proxy.Forward("http://auth-service:8083/swagger"))

	// Access Booking docs at: http://localhost:8080/docs/booking/index.html
	r.Any("/docs/booking/*any", proxy.Forward("http://booking-service:8084/swagger"))

	// Access User docs at: http://localhost:8080/docs/user/index.html
	r.Any("/docs/user/*any", proxy.Forward("http://user-service:8081/swagger"))

}
