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

}
