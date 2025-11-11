package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/controllers"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/auth")
	api.POST("/register", controllers.Register)
	api.POST("/login", controllers.Login)
}
