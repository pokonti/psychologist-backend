package models

import (
	"time"
)

type UserProfile struct {
	ID                  string    `gorm:"primaryKey;type:uuid" json:"id"`
	AuthID              uint      `json:"auth_id"` // reference to auth-service user
	Email               string    `gorm:"unique;not null" json:"email"`
	Role                string    `json:"role"` // e.g. "client", "psychologist", "admin"
	FullName            string    `json:"full_name"`
	Password            string    `json:"password"`
	Phone               string    `json:"phone_number"`
	Gender              string    `json:"gender"`
	Bio                 string    `json:"bio"`
	BirthDate           time.Time `json:"birth_date"`
	Specialization      string    `json:"specialization,omitempty"` // for psychologists
	AvatarURL           string    `json:"avatar_url"`
	Experience          int       `json:"experience,omitempty"`
	Description         string    `json:"description,omitempty"`
	Rating              float32   `json:"rating"`
	NotificationChannel string    `json:"notification_channel"` // "email", "whatsapp", "none"

	CreatedAt time.Time
	UpdatedAt time.Time
}
