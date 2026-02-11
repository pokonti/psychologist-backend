package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/controllers"
)

func SetupRoutes(r *gin.Engine, authController *controllers.AuthController) {
	api := r.Group("/api/v1/auth")

	api.POST("/register", authController.Register)
	api.POST("/login", authController.Login)
}
