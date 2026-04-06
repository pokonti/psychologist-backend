package clients

import (
	"os"

	"github.com/pokonti/psychologist-backend/proto/bookings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewBookingClient() (bookings.BookingServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("BOOKING_SERVICE_GRPC_ADDR")
	if addr == "" {
		addr = "booking-service:9094"
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	return bookings.NewBookingServiceClient(conn), conn, nil
}
