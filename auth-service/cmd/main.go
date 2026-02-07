package main

import (
	"log"

	"github.com/pokonti/psychologist-backend/auth-service/clients"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	config.ConnectDB()

	// gRPC client to user-service
	clients.InitUserProfileClient()

	routes.SetupRoutes(r)

	log.Println("Auth Service running on port 8083")
	r.Run(":8083")
}
