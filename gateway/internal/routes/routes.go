package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/gateway/internal/proxy"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// Public routes
	api.POST("/users/register", proxy.Forward("http://user-service:8081"))
	api.POST("/users/login", proxy.Forward("http://user-service:8081"))

	api.POST("/auth/register", proxy.Forward("http://auth-service:8082"))
	api.POST("/auth/login", proxy.Forward("http://auth-service:8082"))
	// Protected routes
	//protected := api.Group("", middleware.JWTAuth())
	//protected.GET("/psychologists", proxy.Forward("http://psychologist-service:8082"))
	//protected.POST("/appointments", proxy.Forward("http://appointment-service:8083"))
}
