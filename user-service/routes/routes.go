package routes

import (
	"github.com/pokonti/psychologist-backend/user-service/internal/handlers"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, profileHandler *handlers.ProfileHandler) {
	api := r.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.GET("/me", profileHandler.GetMyProfile)
			users.PUT("/me", profileHandler.UpdateMyProfile)
			users.GET("/psychologists", profileHandler.GetPublicPsychologists)
			users.POST("/me/mood", profileHandler.LogMood)
			users.GET("/me/mood/graphic", profileHandler.GetMoodGraphic)
			users.POST("/me/avatar-url", profileHandler.GenerateUploadURL)
		}
		admin := api.Group("/admin")
		{
			admin.GET("/users", profileHandler.ListAllUsers)
			admin.GET("/psychologists", profileHandler.GetAllPsychologists)
			admin.GET("/users/:id", profileHandler.AdminGetUserDetails)
		}
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
