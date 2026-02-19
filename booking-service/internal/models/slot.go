package models

import (
	"time"
)

type Slot struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	PsychologistID string    `gorm:"type:uuid;not null;index" json:"psychologist_id"`
	StartTime      time.Time `gorm:"not null" json:"start_time"`
	Duration       int       `gorm:"default:50" json:"duration"` // in minutes

	// Booking Info
	IsBooked  bool    `gorm:"default:false" json:"is_booked"`
	StudentID *string `gorm:"type:uuid;default:null" json:"student_id"` // Nullable

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `gorm:"default:1" json:"-"`
}
