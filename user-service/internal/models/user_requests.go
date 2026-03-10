package models

type UpdateProfileRequest struct {
	FullName            *string `json:"full_name" binding:"omitempty"`
	Gender              *string `json:"gender" binding:"omitempty"`
	Specialization      *string `json:"specialization" binding:"omitempty"`
	Bio                 *string `json:"bio" binding:"omitempty"`
	AvatarURL           *string `json:"avatar_url" binding:"omitempty"`
	Phone               *string `json:"phone" binding:"omitempty"`
	NotificationChannel *string `json:"notification_channel" binding:"omitempty"`
}
