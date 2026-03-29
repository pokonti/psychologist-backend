package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/internal/clients"
	"github.com/pokonti/psychologist-backend/auth-service/internal/handlers"
	"github.com/pokonti/psychologist-backend/auth-service/internal/routes"

	_ "github.com/pokonti/psychologist-backend/auth-service/docs"
)

// @title           Auth Service API
// @version         1.0
// @description     This is the authentication service for the Psychologist Backend.
// @host            localhost:8080
// @BasePath        /api/v1/auth
func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.TrustedPlatform = gin.PlatformCloudflare
	config.ConnectDB()

	config.ConnectRabbitMQ()
	defer config.RabbitConn.Close()
	defer config.RabbitChannel.Close()

	// Init gRPC Client
	userClient, conn, err := clients.NewUserProfileClient()
	if err != nil {
		log.Fatalf("Could not connect to user service: %v", err)
	}
	defer conn.Close()

	rabbitMQ := clients.NewRabbitMQClient()
	// Init Controller
	authController := &handlers.AuthController{
		UserClient: userClient,
		RabbitMQ:   rabbitMQ,
	}

	routes.SetupRoutes(r, authController)

	log.Println("Auth Service running on port 8083")
	r.Run(":8083")
}
