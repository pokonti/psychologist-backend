package models

import "time"

type SlotResponse struct {
	ID               string    `json:"id"`
	StartTime        time.Time `json:"start_time"`
	Duration         int       `json:"duration"`
	Status           string    `json:"status"`
	BookingType      string    `json:"booking_type"`
	PsychologistID   string    `json:"psychologist_id"`
	PsychologistName string    `json:"psychologist_name"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid start_date"`
}
type MessageResponse struct {
	Message string `json:"message" example:"Booking successful"`
}

// ScheduleCreatedResponse represents the result of schedule generation.
type ScheduleCreatedResponse struct {
	Message string `json:"message" example:"Schedule created successfully"`
	Count   int    `json:"count" example:"24"`
}

// CalendarAvailabilityResponse represents available days in a month.
type CalendarAvailabilityResponse struct {
	AvailableDates []string `json:"available_dates" example:"2026-02-10,2026-02-14"`
}

type PsychologistScheduleResponse struct {
	ID                   string    `json:"id"`
	StartTime            time.Time `json:"start_time"`
	Duration             int       `json:"duration"`
	Status               string    `json:"status"`
	BookingType          string    `json:"booking_type"`
	PsychologistID       string    `json:"psychologist_id"`
	StudentID            *string   `json:"student_id,omitempty"`
	StudentName          string    `json:"student_name"`
	QuestionnaireAnswers string    `json:"questionnaire_answers,omitempty"`
	PhoneNumber          string    `json:"phone_number,omitempty"`
}

type StudentAppointmentResponse struct {
	ID                     string    `json:"id"`
	StartTime              time.Time `json:"start_time"`
	Duration               int       `json:"duration"`
	BookingType            string    `json:"booking_type"`
	PsychologistID         string    `json:"psychologist_id"`
	PsychologistName       string    `json:"psychologist_name"`
	QuestionnaireAnswers   string    `json:"questionnaire_answers,omitempty"`
	StudentRecommendations string    `json:"student_recommendations,omitempty"`
}

type StudentHistoryResponse struct {
	SlotID               string    `json:"slot_id"`
	StartTime            time.Time `json:"start_time"`
	BookingType          string    `json:"booking_type"`
	QuestionnaireAnswers string    `json:"questionnaire_answers"`
	PsychologistNotes    string    `json:"psychologist_notes"`
}
type WaitlistResponse struct {
	ID               string    `json:"id"`
	PsychologistID   string    `json:"psychologist_id"`
	PsychologistName string    `json:"psychologist_name"`
	Date             string    `json:"date"`
	CreatedAt        time.Time `json:"created_at"`
}
