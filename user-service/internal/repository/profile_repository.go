package repository

import (
	"context"

	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(ctx context.Context, p *models.UserProfile) error
	GetByID(ctx context.Context, id string) (*models.UserProfile, error)
	Update(ctx context.Context, p *models.UserProfile) error
}
type GormProfileRepository struct {
	db *gorm.DB
}

func NewGormProfileRepository(db *gorm.DB) *GormProfileRepository {
	return &GormProfileRepository{db: db}
}

func (r *GormProfileRepository) Create(ctx context.Context, p *models.UserProfile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *GormProfileRepository) GetByID(ctx context.Context, id string) (*models.UserProfile, error) {
	var profile models.UserProfile
	if err := r.db.WithContext(ctx).First(&profile, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *GormProfileRepository) Update(ctx context.Context, p *models.UserProfile) error {
	return r.db.WithContext(ctx).Save(p).Error
}
