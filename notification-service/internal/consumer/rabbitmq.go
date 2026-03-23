package consumer

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pokonti/psychologist-backend/notification-service/internal/email"
	"github.com/pokonti/psychologist-backend/notification-service/internal/models"
	"github.com/pokonti/psychologist-backend/notification-service/internal/telegram"
	amqp "github.com/rabbitmq/amqp091-go"
)

// StartListening now takes the channel and queue injected from the config
func StartListening(ch *amqp.Channel, q amqp.Queue) {
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	log.Println("Notification Service is waiting for messages.")

	// Process Messages in a continuous loop
	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			processMessage(d.Body)
		}
	}()

	<-forever // Blocks the main thread forever
}

// processMessage remains exactly the same as your code
func processMessage(body []byte) {
	var msg models.NotificationMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	var subject, htmlBody string

	switch msg.Type {
	case "booking_confirmation":
		subject = "Appointment Confirmed! ✅"
		htmlBody = fmt.Sprintf(`
			<h2>Your Appointment is Confirmed</h2>
			<p>You have successfully booked a session.</p>
			<ul>
				<li><b>Specialist:</b> %s</li>
				<li><b>Date & Time:</b> %s</li>
				<li><b>Format:</b> %s</li>
			</ul>
			<p>Thank you for using KBTU Care.</p>
		`, msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["format"])

	case "auth_verification":
		subject = "Verify your KBTU Care Account"
		htmlBody = fmt.Sprintf(`
			<h2>Welcome to KBTU Care!</h2>
			<p>Your verification code is: <b>%s</b></p>
			<p>This code will expire in 15 minutes.</p>
		`, msg.Data["code"])

	case "booking_cancellation":
		subject = "Appointment Canceled ❌"
		htmlBody = fmt.Sprintf(`
			<h2>Appointment Canceled</h2>
			<p>Your appointment with <b>%s</b> on <b>%s</b> has been canceled.</p>
			<p>We hope to see you again soon.</p>`, msg.Data["psychologist_name"], msg.Data["datetime"])

	case "booking_cancellation_by_psychologist":
		subject = "Appointment Canceled ❌"
		htmlBody = fmt.Sprintf(`
			<h2>Appointment Canceled</h2>
			<p>Hello,</p>
			<p>Your appointment with <b>%s</b> scheduled for <b>%s</b> has been canceled by <b>%s</b>.</p>
			<p>We apologize for the inconvenience.</p>
			<p>If you have any questions, please contact the KBTU Care support team.</p>
		`, msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["psychologist_name"])

	case "booking_reschedule":
		subject = "Appointment Rescheduled 📅"
		htmlBody = fmt.Sprintf(`
			<h2>Appointment Rescheduled</h2>
			<p>Your appointment has been successfully moved.</p>
			<ul>
				<li><b>Specialist:</b> %s</li>
				<li><b>New Date & Time:</b> %s</li>
				<li><b>Format:</b> %s</li>
			</ul>`, msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["format"])

	case "waitlist_alert":
		subject = "A Slot Just Opened Up! 🚨"
		htmlBody = fmt.Sprintf(`
			<h2>Good News!</h2>
			<p>A time slot has just become available for <b>%s</b> on <b>%s</b>.</p>
			<p>Since you are on the waitlist, we are letting you know immediately.</p>
			<p>Please log in to the KBTU Care platform to book this slot before someone else takes it!</p>
			<br>
			<p><i>Note: Waitlist alerts are sent to everyone waiting for this day. Booking is first-come, first-served.</i></p>
		`, msg.Data["psychologist_name"], msg.Data["date"])
	case "new_recommendation":
		subject = "Post-Session Recommendations 📝"
		htmlBody = fmt.Sprintf(`
			<h2>Recommendations from %s</h2>
			<p>Your psychologist has shared some notes and recommendations from your session on <b>%s</b>.</p>
			<p>Please log in to your KBTU Care dashboard and visit "My Appointments" to view them.</p>
			<br>
			<p>Take care,<br>The KBTU Care Team</p>
		`, msg.Data["psychologist_name"], msg.Data["date"])
	case "account_blocked":
		subject = "Your Account Access Status ⚠️"
		htmlBody = fmt.Sprintf(`
			<h2>Account Notice</h2>
			<p>Your KBTU Care account has been restricted.</p>
			<p><b>Reason:</b> %s</p>
			<p>If you believe this is a mistake, please contact the administration office.</p>
		`, msg.Data["reason"])
	case "session_reminder":
		subject = "Reminder: Upcoming Appointment ⏰"
		htmlBody = fmt.Sprintf(`
			<h2>Appointment Reminder</h2>
			<p>You have a session with <b>%s</b>  <b>%s</b> at <b>%s</b>.</p>
			<p>Please ensure you are on time!</p>
		`, msg.Data["psychologist_name"], msg.Data["subject"], msg.Data["datetime"])

		tgChatID := msg.Data["telegram_chat_id"]
		if tgChatID != "" {
			tgText := fmt.Sprintf("⏰ <b>Reminder!</b>\nYou have an appointment with %s tomorrow at %s.", msg.Data["psychologist_name"], msg.Data["datetime"])
			telegram.SendMessage(tgChatID, tgText)
		}

	default:
		log.Printf("Unknown message type: %s", msg.Type)
		return
	}

	log.Printf("Sending email to %s...", msg.ToEmail)
	err := email.SendEmail(msg.ToEmail, subject, htmlBody)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", msg.ToEmail, err)
	} else {
		log.Printf("Email sent successfully to %s", msg.ToEmail)
	}
}
