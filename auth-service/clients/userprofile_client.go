package clients

import (
	"log"
	"os"

	"github.com/pokonti/psychologist-backend/proto/userprofile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var UserProfileClient userprofile.UserProfileServiceClient

func InitUserProfileClient() {
	addr := os.Getenv("USER_SERVICE_GRPC_ADDR")
	if addr == "" {
		addr = "user-service:9091" // docker-compose service name + gRPC port
	}

	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to user-service gRPC: %v", err)
	}

	UserProfileClient = userprofile.NewUserProfileServiceClient(conn)
	log.Printf("Connected to user-service gRPC at %s", addr)
}
