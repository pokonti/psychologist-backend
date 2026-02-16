package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/internal/handlers"
)

func SetupRoutes(r *gin.Engine, authController *handlers.AuthController) {
	api := r.Group("/api/v1/auth")

	api.POST("/register", authController.Register)
	api.POST("/verify", authController.VerifyEmail)
	api.POST("/login", authController.Login)
}
