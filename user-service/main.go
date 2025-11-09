package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/models"
	"github.com/pokonti/psychologist-backend/user-service/routes"
)

func main() {
	config.ConnectDB()
	config.DB.AutoMigrate(&models.User{})

	r := gin.Default()
	routes.SetupRoutes(r)
	r.Run(":8081")
}
