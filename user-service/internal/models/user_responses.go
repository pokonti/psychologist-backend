package models

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid start_date"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type UploadResponse struct {
	UploadURL string `json:"upload_url"` // Frontend uses this to PUT the file
	FinalURL  string `json:"final_url"`  // Frontend saves this to the user profile
}
