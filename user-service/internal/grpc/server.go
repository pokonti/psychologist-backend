package grpcserver

import (
	"context"
	"errors"

	"github.com/pokonti/psychologist-backend/proto/userprofile"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"github.com/pokonti/psychologist-backend/user-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gorm.io/gorm"
)

type UserProfileServer struct {
	userprofile.UnimplementedUserProfileServiceServer
	Repo repository.ProfileRepository
}

func NewUserProfileServer(repo repository.ProfileRepository) *UserProfileServer {
	return &UserProfileServer{Repo: repo}
}

func (s *UserProfileServer) CreateUserProfile(ctx context.Context, req *userprofile.CreateUserProfileRequest) (*userprofile.CreateUserProfileResponse, error) {
	profile := &models.UserProfile{
		ID:       req.Id,
		Email:    req.Email,
		Role:     req.Role,
		FullName: req.FullName,
	}

	// If profile already exists, you can choose to update or ignore
	if err := s.Repo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return &userprofile.CreateUserProfileResponse{Id: profile.ID}, nil
}

func (s *UserProfileServer) GetUserProfileByID(ctx context.Context, req *userprofile.GetUserProfileByIDRequest) (*userprofile.GetUserProfileByIDResponse, error) {
	p, err := s.Repo.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "profile not found")
		}
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &userprofile.GetUserProfileByIDResponse{
		Id:             p.ID,
		FullName:       p.FullName,
		Gender:         p.Gender,
		Role:           p.Role,
		Specialization: p.Specialization,
		Email:          p.Email,
		Phone:          p.Phone,
		TelegramChatId: p.TelegramChatID,
	}, nil
}

func (s *UserProfileServer) GetBatchUserProfiles(ctx context.Context, req *userprofile.GetBatchUserProfilesRequest) (*userprofile.GetBatchUserProfilesResponse, error) {
	if len(req.Ids) == 0 {
		return &userprofile.GetBatchUserProfilesResponse{}, nil
	}

	// Call the new Repository method
	users, err := s.Repo.GetByIDs(ctx, req.Ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	// Map DB models to Proto models
	var profiles []*userprofile.BasicUserProfile
	for _, u := range users {
		profiles = append(profiles, &userprofile.BasicUserProfile{
			Id:             u.ID,
			FullName:       u.FullName,
			Email:          u.Email,
			TelegramChatId: u.TelegramChatID,
		})
	}

	return &userprofile.GetBatchUserProfilesResponse{Profiles: profiles}, nil
}

func (s *UserProfileServer) UpdateUserPhone(ctx context.Context, req *userprofile.UpdateUserPhoneRequest) (*userprofile.UpdateUserPhoneResponse, error) {
	var profile models.UserProfile
	if err, _ := s.Repo.GetByID(ctx, req.Id); err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	profile.Phone = req.Phone
	if err := s.Repo.Update(ctx, &profile); err != nil {
		return nil, status.Error(codes.Internal, "failed to update phone")
	}

	return &userprofile.UpdateUserPhoneResponse{Success: true}, nil
}

func (s *UserProfileServer) UpdateUserTelegram(ctx context.Context, req *userprofile.UpdateUserTelegramRequest) (*userprofile.UpdateUserTelegramResponse, error) {
	var profile models.UserProfile
	if err, _ := s.Repo.GetByID(ctx, req.Id); err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	profile.TelegramChatID = req.TelegramChatId
	if err := s.Repo.Update(ctx, &profile); err != nil {
		return nil, status.Error(codes.Internal, "failed to update telegram chat id")
	}

	return &userprofile.UpdateUserTelegramResponse{Success: true}, nil
}

func (s *UserProfileServer) UpdateUserBlockStatus(ctx context.Context, req *userprofile.UpdateUserBlockStatusRequest) (*userprofile.UpdateUserBlockStatusResponse, error) {
	var profile models.UserProfile
	if err, _ := s.Repo.GetByID(ctx, req.Id); err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	profile.IsBlocked = req.IsBlocked
	profile.BlockReason = req.Reason

	if err := s.Repo.Update(ctx, &profile); err != nil {
		return nil, status.Error(codes.Internal, "failed to update block status")
	}

	return &userprofile.UpdateUserBlockStatusResponse{Success: true}, nil
}
