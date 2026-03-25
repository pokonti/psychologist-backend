package models

type UpdateProfileRequest struct {
	FullName       *string `json:"full_name" binding:"omitempty"`
	Gender         *string `json:"gender" binding:"omitempty"`
	Specialization *string `json:"specialization" binding:"omitempty"`
	Bio            *string `json:"bio" binding:"omitempty"`
	AvatarURL      *string `json:"avatar_url" binding:"omitempty"`
	Phone          *string `json:"phone" binding:"omitempty"`
}

// PublicPsychologistResponse represents the safe public profile of a psychologist
type PublicPsychologistResponse struct {
	ID             string  `json:"id"`
	FullName       string  `json:"full_name"`
	Gender         string  `json:"gender"`
	Bio            string  `json:"bio"`
	Specialization string  `json:"specialization,omitempty"`
	AvatarURL      string  `json:"avatar_url"`
	Experience     int     `json:"experience,omitempty"`
	Description    string  `json:"description,omitempty"`
	Rating         float32 `json:"rating"`
}

type UploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`    // e.g. "avatar.png"
	ContentType string `json:"content_type" binding:"required"` // e.g. "image/png"
}
