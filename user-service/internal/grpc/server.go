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
	}, nil
}

func (s *UserProfileServer) GetBatchUserProfiles(ctx context.Context, req *userprofile.GetBatchUserProfilesRequest) (*userprofile.GetBatchUserProfilesResponse, error) {
	if len(req.Ids) == 0 {
		return &userprofile.GetBatchUserProfilesResponse{}, nil
	}

	// 1. Call the new Repository method
	users, err := s.Repo.GetByIDs(ctx, req.Ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	// 2. Map DB models to Proto models
	var profiles []*userprofile.BasicUserProfile
	for _, u := range users {
		profiles = append(profiles, &userprofile.BasicUserProfile{
			Id:       u.ID,
			FullName: u.FullName,
		})
	}

	return &userprofile.GetBatchUserProfilesResponse{Profiles: profiles}, nil
}
