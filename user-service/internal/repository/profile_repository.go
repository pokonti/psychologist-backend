package repository

import (
	"context"

	"github.com/pokonti/psychologist-backend/user-service/internal/models"
)

type ProfileRepository interface {
	Create(ctx context.Context, profile *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, profile *models.User) error
}
