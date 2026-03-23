package models

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid start_date"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
