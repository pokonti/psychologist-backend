package main

import (
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
	"github.com/pokonti/psychologist-backend/user-service/config"
	grpcserver "github.com/pokonti/psychologist-backend/user-service/internal/grpc"
	"github.com/pokonti/psychologist-backend/user-service/internal/handlers"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"github.com/pokonti/psychologist-backend/user-service/internal/repository"
	"github.com/pokonti/psychologist-backend/user-service/routes"
	"google.golang.org/grpc"

	_ "github.com/pokonti/psychologist-backend/user-service/docs"
)

// @title           User Service API
// @version         1.0
// @description     User profiles and psychologist directory for KBTU counseling system.
// @host            localhost:8081
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	config.ConnectDB()
	db := config.DB

	if err := db.AutoMigrate(&models.UserProfile{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	profileRepo := repository.NewGormProfileRepository(db)

	go func() {
		r := gin.Default()
		profileHandler := handlers.NewProfileHandler(profileRepo)
		routes.SetupRoutes(r, profileHandler)

		log.Println("user-service HTTP listening on :8081")
		if err := r.Run(":8081"); err != nil {
			log.Fatal(err)
		}
	}()

	// gRPC server (for other services)
	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	userprofile.RegisterUserProfileServiceServer(
		grpcServer,
		grpcserver.NewUserProfileServer(profileRepo),
	)

	log.Println("user-service gRPC listening on :9091")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
