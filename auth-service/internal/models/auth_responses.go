package models

type RegisterResponse struct {
	Message string `json:"message"`
}

type TokenResponse struct {
	Token   string `json:"token"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
