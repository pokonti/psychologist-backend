package routes

import (
	"github.com/pokonti/psychologist-backend/user-service/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, profileHandler *handlers.ProfileHandler) {
	r.GET("/users/me", profileHandler.GetMyProfile)

}
