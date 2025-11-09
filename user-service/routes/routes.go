package routes

import (
	"github.com/pokonti/psychologist-backend/user-service/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/users")
	api.POST("/register", controllers.Register)
	api.POST("/login", controllers.Login)
}
