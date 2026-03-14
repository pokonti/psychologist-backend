package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/internal/handlers"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(r *gin.Engine, authController *handlers.AuthController) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1/auth")
	api.POST("/register", authController.Register)
	api.POST("/verify", authController.VerifyEmail)
	api.POST("/login", authController.Login)
	api.POST("/refresh", authController.RefreshToken)
	api.POST("/logout", authController.Logout)

	admin := r.Group("/api/v1/admin")
	{
		admin.POST("/users", authController.AdminAddUser)
		admin.PATCH("/users/:id/block", authController.AdminBlockUser)
	}
}
