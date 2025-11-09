package gateway

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/gateway/internal/routes"
)

func main() {
	r := gin.Default()

	// Setup all routes
	routes.SetupRoutes(r)

	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API Gateway running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
