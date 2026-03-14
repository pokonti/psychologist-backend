package models

type NotificationMessage struct {
	Type    string            `json:"type"`     // e.g., "booking_confirmation", "auth_verification"
	ToEmail string            `json:"to_email"` // Who receives the email
	Data    map[string]string `json:"data"`     // Dynamic data (names, dates, links)
}
