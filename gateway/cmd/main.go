package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/gateway/internal/middleware"
	"github.com/pokonti/psychologist-backend/gateway/internal/routes"
)

func main() {
	r := gin.Default()

	r.TrustedPlatform = gin.PlatformCloudflare

	routes.SetupRoutes(r)

	r.Use(middleware.CorsMiddleware())

	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API Gateway running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
