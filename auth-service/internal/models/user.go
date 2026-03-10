package models

import "time"

type User struct {
	ID       string `gorm:"primaryKey;type:uuid" json:"id"`
	Email    string `gorm:"uniqueIndex;not null" json:"email"`
	Password string `gorm:"not null" json:"-"`
	Role     string `gorm:"not null" json:"role"` // student / psychologist / admin

	VerificationCode string    `json:"-"`
	CodeExpiresAt    time.Time `json:"-"`
	IsVerified       bool      `gorm:"default:false" json:"is_verified"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
