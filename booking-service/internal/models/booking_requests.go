package models

type CreateScheduleInput struct {
	StartDate string        `json:"start_date" binding:"required"` // "2026-02-20"
	EndDate   string        `json:"end_date" binding:"required"`   // "2026-03-20"
	Duration  int           `json:"duration"`                      // 50 (default)
	Schedule  []DaySchedule `json:"schedule" binding:"required"`   // The template
}

type DaySchedule struct {
	DayOfWeek  int      `json:"day_of_week" binding:"required"` // 1=Mon, 2=Tue...
	StartTimes []string `json:"start_times" binding:"required"` // ["09:00", "10:00", "14:00"]
}

type BookSlotInput struct {
	BookingType string `json:"booking_type" binding:"required,oneof=online offline"`
	Answers     string `json:"answers"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type RescheduleInput struct {
	NewSlotID string `json:"new_slot_id" binding:"required"`
}

type AddNoteInput struct {
	Notes string `json:"notes" binding:"required"`
}

type JoinWaitlistInput struct {
	PsychologistID string `json:"psychologist_id" binding:"required"`
	Date           string `json:"date" binding:"required"`
}

type RecommendationInput struct {
	Recommendations string `json:"recommendations" binding:"required"`
}

type RateSessionInput struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"` // Must be 1-5
	Review string `json:"review" binding:"omitempty,max=500"`    // Optional text review
}
