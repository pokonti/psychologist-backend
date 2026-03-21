package models

import (
	"time"
)

const (
	StatusAvailable = "available"
	StatusReserved  = "reserved"
	StatusBooked    = "booked"
)

type Slot struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	PsychologistID string    `gorm:"type:uuid;not null;uniqueIndex:idx_psych_time" json:"psychologist_id"`
	StartTime      time.Time `gorm:"not null;uniqueIndex:idx_psych_time" json:"start_time"`
	Duration       int       `gorm:"default:50" json:"duration"` // in minutes

	Status     string     `gorm:"default:'available';index" json:"status"`  // available, reserved, booked
	ReservedAt *time.Time `json:"reserved_at,omitempty"`                    // When the 20-min lock started
	StudentID  *string    `gorm:"type:uuid;default:null" json:"student_id"` // Nullable

	BookingType string `gorm:"default:null" json:"booking_type"` // "online" or "offline"

	QuestionnaireAnswers string `gorm:"type:text" json:"questionnaire_answers"`

	PsychologistNotes string `gorm:"type:text" json:"psychologist_notes,omitempty"`

	StudentRecommendations string `gorm:"type:text" json:"student_recommendations,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `gorm:"default:1" json:"-"`
}
