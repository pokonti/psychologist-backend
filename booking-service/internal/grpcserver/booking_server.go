package grpcserver

import (
	"context"

	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/bookings"
)

type BookingGrpcServer struct {
	bookings.UnimplementedBookingServiceServer
}

func (s *BookingGrpcServer) UpdateMeetingLink(ctx context.Context, req *bookings.UpdateMeetingLinkRequest) (*bookings.UpdateMeetingLinkResponse, error) {
	err := config.DB.Model(&models.Slot{}).
		Where("id = ?", req.SlotId).
		Update("meeting_link", req.MeetingLink).Error

	if err != nil {
		return &bookings.UpdateMeetingLinkResponse{Success: false}, err
	}

	return &bookings.UpdateMeetingLinkResponse{Success: true}, nil
}
