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
	protected.GET("/slots", proxy.Forward("http://booking-service:8084"))
	psychOnly := protected.Group("/psychologist", middleware.RequireRoles("psychologist", "admin"))
	{
		psychOnly.POST("/slots", proxy.Forward("http://booking-service:8084"))
	}
	//// Routes for ADMINS ONLY
	//adminOnly := protected.Group("/", middleware.RequireRoles("admin"))
	//{
	//	// Example: Delete users, view analytics
	//	adminOnly.DELETE("/users/:id", proxy.Forward("http://user-service:8081"))
	//	adminOnly.GET("/analytics", proxy.Forward("http://analytics-service:8087"))
	//}
	//
	// Routes for STUDENTS ONLY
	studentOnly := protected.Group("/student", middleware.RequireRoles("student", "admin"))
	{
		// Book an appointment
		studentOnly.POST("/slots/:id/book", proxy.Forward("http://booking-service:8084"))
	}

}
