package models

import (
	"time"
)

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	AuthID         uint      `json:"auth_id"` // reference to auth-service user
	Email          string    `gorm:"unique;not null" json:"email"`
	Role           string    `json:"role"` // e.g. "client", "psychologist", "admin"
	FullName       string    `json:"full_name"`
	Gender         string    `json:"gender"`
	BirthDate      time.Time `json:"birth_date"`
	Specialization string    `json:"specialization,omitempty"` // for psychologists
	Experience     int       `json:"experience,omitempty"`
	Description    string    `json:"description,omitempty"`
	Rating         float32   `json:"rating"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
