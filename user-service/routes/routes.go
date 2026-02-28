package routes

import (
	"github.com/pokonti/psychologist-backend/user-service/internal/handlers"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, profileHandler *handlers.ProfileHandler) {
	api := r.Group("/api/v1/users")
	{
		api.GET("/me", profileHandler.GetMyProfile)
		api.PUT("/me", profileHandler.UpdateMyProfile)
		api.GET("/psychologists", profileHandler.GetAllPsychologists)
	}
	// Swagger Route
	// Access at: http://localhost:8081/swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
