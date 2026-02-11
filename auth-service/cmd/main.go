package main

import (
	"log"

	"github.com/pokonti/psychologist-backend/auth-service/clients"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/controllers"
	"github.com/pokonti/psychologist-backend/auth-service/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	config.ConnectDB()

	// Init gRPC Client
	userClient, conn, err := clients.NewUserProfileClient()
	if err != nil {
		log.Fatalf("Could not connect to user service: %v", err)
	}
	defer conn.Close() // Setup graceful shutdown

	// Init Controller
	authController := &controllers.AuthController{
		UserClient: userClient,
	}

	routes.SetupRoutes(r, authController)

	log.Println("Auth Service running on port 8083")
	r.Run(":8083")
}
