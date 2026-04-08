package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/internal/handlers"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(r *gin.Engine, authController *handlers.AuthController) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/verify", authController.VerifyEmail)
			auth.POST("/login", authController.Login)
			auth.POST("/refresh", authController.RefreshToken)
			auth.POST("/logout", authController.Logout)
		}

		admin := api.Group("/admin")
		{
			admin.POST("/users", authController.AdminAddUser)
			admin.PATCH("/users/:id/block", authController.AdminBlockUser)
		}
	}
}
