package models

import (
	"time"
)

type MoodLog struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_user_date" json:"user_id"`
	Date      time.Time `gorm:"type:date;not null;uniqueIndex:idx_user_date" json:"date"` // Format: YYYY-MM-DD
	Mood      string    `gorm:"not null" json:"mood"`
	Score     int       `gorm:"not null" json:"score"` // 1 to 6 (for the graph)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogMoodInput struct {
	Mood string `json:"mood" binding:"required,oneof=Amazing Nice 'Not bad' Sad Anxiously Stressed"`
}

type MoodGraphicResponse struct {
	Date      string `json:"date"`        // "2026-02-23"
	DayOfWeek string `json:"day_of_week"` // "Mon", "Tue"
	Mood      string `json:"mood"`        // "Amazing"
	Score     int    `json:"score"`       // 6
}
