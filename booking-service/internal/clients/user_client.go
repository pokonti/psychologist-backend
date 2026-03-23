package clients

import (
	"log"
	"os"

	"github.com/pokonti/psychologist-backend/proto/userprofile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewUserProfileClient() (userprofile.UserProfileServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("USER_SERVICE_GRPC_ADDR")
	if addr == "" {
		addr = "user-service:9091"
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	log.Printf("Booking Service connected to User Service at %s", addr)
	return userprofile.NewUserProfileServiceClient(conn), conn, nil
}
