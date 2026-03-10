package models

import "time"

type WaitlistEntry struct {
	ID             string `gorm:"type:uuid;primary_key" json:"id"`
	StudentID      string `gorm:"type:uuid;not null;uniqueIndex:idx_waitlist" json:"student_id"`
	PsychologistID string `gorm:"type:uuid;not null;uniqueIndex:idx_waitlist" json:"psychologist_id"`

	Date string `gorm:"not null;uniqueIndex:idx_waitlist" json:"date"`

	CreatedAt time.Time `json:"created_at"`
}
