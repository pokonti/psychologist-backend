package models

import (
	"time"
)

type Slot struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	PsychologistID string    `gorm:"type:uuid;not null;index" json:"psychologist_id"`
	StartTime      time.Time `gorm:"not null;index" json:"start_time"`
	Duration       int       `gorm:"default:50" json:"duration"` // in minutes

	// Booking Info
	IsBooked  bool    `gorm:"default:false" json:"is_booked"`
	StudentID *string `gorm:"type:uuid;default:null" json:"student_id"` // Nullable

	BookingType string `gorm:"default:null" json:"booking_type"` // "online" or "offline"

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `gorm:"default:1" json:"-"`
}

type DaySchedule struct {
	DayOfWeek  int      `json:"day_of_week" binding:"required"` // 1=Mon, 2=Tue...
	StartTimes []string `json:"start_times" binding:"required"` // ["09:00", "10:00", "14:00"]
}

type CreateScheduleInput struct {
	StartDate string        `json:"start_date" binding:"required"` // "2026-02-20"
	EndDate   string        `json:"end_date" binding:"required"`   // "2026-03-20"
	Duration  int           `json:"duration"`                      // 50 (default)
	Schedule  []DaySchedule `json:"schedule" binding:"required"`   // The template
}

type SlotResponse struct {
	ID               string    `json:"id"`
	StartTime        time.Time `json:"start_time"`
	Duration         int       `json:"duration"`
	IsBooked         bool      `json:"is_booked"`
	BookingType      string    `json:"booking_type"`
	PsychologistID   string    `json:"psychologist_id"`
	PsychologistName string    `json:"psychologist_name"` // Enriched via gRPC
}
