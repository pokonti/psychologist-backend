package repository

import (
	"context"
	"database/sql"

	"github.com/pokonti/psychologist-backend/user-service/internal/models"
)

type PostgresProfileRepository struct {
	db *sql.DB
}

func NewPostgresProfileRepository(db *sql.DB) *PostgresProfileRepository {
	return &PostgresProfileRepository{db: db}
}

func (r *PostgresProfileRepository) Create(ctx context.Context, p *models.User) error {
	query := `
        INSERT INTO user_profiles (
            id, full_name, gender, specialization,
            bio, avatar_url, phone, city
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := r.db.ExecContext(
		ctx,
		query,
		p.ID,
		p.FullName,
		p.Gender,
		p.Specialization,
		p.Bio,
		p.AvatarURL,
		p.Phone,
		p.City,
	)

	return err
}

func (r *PostgresProfileRepository) GetByID(ctx context.Context, id string) (*models.UserProfile, error) {
	query := `
        SELECT id, full_name, gender, specialization,
               bio, avatar_url, phone, city
        FROM user_profiles
        WHERE id = $1
    `

	var p models.UserProfile

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.FullName,
		&p.Gender,
		&p.Specialization,
		&p.Bio,
		&p.AvatarURL,
		&p.Phone,
		&p.City,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PostgresProfileRepository) Update(ctx context.Context, p *models.UserProfile) error {
	query := `
        UPDATE user_profiles
        SET full_name = $2,
            gender = $3,
            specialization = $4,
            bio = $5,
            avatar_url = $6,
            phone = $7,
            city = $8
        WHERE id = $1
    `

	_, err := r.db.ExecContext(
		ctx,
		query,
		p.ID,
		p.FullName,
		p.Gender,
		p.Specialization,
		p.Bio,
		p.AvatarURL,
		p.Phone,
		p.City,
	)

	return err
}
