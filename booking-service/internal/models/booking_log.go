package models

import "time"

type BookingLog struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	SlotID         string    `gorm:"type:uuid;index" json:"slot_id"`
	PsychologistID string    `gorm:"type:uuid;index" json:"psychologist_id"`
	StudentID      string    `gorm:"type:uuid;index" json:"student_id"`
	Action         string    `gorm:"type:varchar(50);not null" json:"action"` // "booked", "canceled_by_student", "canceled_by_psychologist", "rescheduled"
	Reason         string    `json:"reason"`
	Timestamp      time.Time `gorm:"index" json:"timestamp"`
}
