package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pokonti/psychologist-backend/notification-service/internal/email"
	"github.com/pokonti/psychologist-backend/notification-service/internal/meetings"
	"github.com/pokonti/psychologist-backend/notification-service/internal/models"
	"github.com/pokonti/psychologist-backend/notification-service/internal/telegram"
	"github.com/pokonti/psychologist-backend/proto/bookings"
	amqp "github.com/rabbitmq/amqp091-go"
)

// StartListening now takes the channel and queue injected from the config
func StartListening(ch *amqp.Channel, q amqp.Queue, bookingClient bookings.BookingServiceClient) {
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
			processMessage(d.Body, bookingClient)
		}
	}()

	<-forever // Blocks the main thread forever
}

func processMessage(body []byte, bookingClient bookings.BookingServiceClient) {
	var msg models.NotificationMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	var subject, htmlBody string

	meetingLink := ""
	if msg.Type == "booking_confirmation" && msg.Data["format"] == "online" {
		zoom := meetings.NewZoomClient()

		startTime, _ := time.Parse(time.RFC3339, msg.Data["start_time_raw"])

		topic := fmt.Sprintf("Session: %s", msg.Data["psychologist_name"])
		link, err := zoom.CreateMeeting(topic, startTime)
		if err == nil {
			meetingLink = link
			_, err := bookingClient.UpdateMeetingLink(context.Background(), &bookings.UpdateMeetingLinkRequest{
				SlotId:      msg.Data["slot_id"],
				MeetingLink: link,
			})
			if err != nil {
				log.Printf("gRPC sync failed: %v", err)
			}
			log.Printf("Zoom link created: %s", link)
		} else {
			log.Printf("Zoom creation failed: %v", err)
		}
	}

	linkHTML := ""
	if meetingLink != "" {
		linkHTML = fmt.Sprintf(`<p><b>Zoom Meeting Link:</b> <a href="%s">Join Session</a></p>`, meetingLink)
	}

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
				%s
			</ul>
			<p>Thank you for using KBTU Care.</p>
		`, msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["format"], linkHTML)

		if tgChatID := msg.Data["telegram_chat_id"]; tgChatID != "" {
			tgText := fmt.Sprintf("✅ <b>Booking Confirmed!</b>\nSpecialist: %s\nTime: %s\nFormat: %s",
				msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["format"])
			err := telegram.SendMessage(tgChatID, tgText)
			if err != nil {
				log.Printf("Failed to send Telegram reminder: %v", err)
			}
		}

	case "auth_verification":
		subject = "Verify your KBTU Care Account"
		htmlBody = fmt.Sprintf(`
			<h2>Welcome to KBTU Care!</h2>
			<p>Your verification code is: <b>%s</b></p>
			<p>This code will expire in 15 minutes.</p>
		`, msg.Data["code"])
	case "booking_cancellation":
		subject = "Appointment Canceled ❌"

		cancelledBy := msg.Data["cancelled_by"]
		reasonTopic := msg.Data["reason_topic"]
		reasonMessage := msg.Data["reason_message"]

		reasonHTML := ""
		if reasonTopic != "" {
			reasonHTML += fmt.Sprintf("<p><b>Reason:</b> %s</p>", reasonTopic)
		}
		if reasonMessage != "" {
			reasonHTML += fmt.Sprintf("<p><i>\"%s\"</i></p>", reasonMessage)
		}

		htmlBody = fmt.Sprintf(`
			<h2>Appointment Canceled</h2>
			<p>Hello,</p>
			<p>Your appointment with <b>%s</b> scheduled for <b>%s</b> has been canceled by <b>%s</b>.</p>
			%s <!-- This inserts the reason block we built above -->
			<p>We apologize for any inconvenience.</p>
			<p>If you have any questions, please contact the KBTU Care support team or book a new slot on the platform.</p>
		`, msg.Data["psychologist_name"], msg.Data["datetime"], cancelledBy, reasonHTML)
	case "booking_cancellation_by_psychologist":
		subject = "Appointment Canceled ❌"
		htmlBody = fmt.Sprintf(`
			<h2>Appointment Canceled</h2>
			<p>Hello,</p>
			<p>Your appointment with <b>%s</b> scheduled for <b>%s</b> has been canceled by <b>%s</b>.</p>
			<p>We apologize for the inconvenience.</p>
			<p>If you have any questions, please contact the KBTU Care support team.</p>
		`, msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["psychologist_name"])

		if tgChatID := msg.Data["telegram_chat_id"]; tgChatID != "" {
			tgText := fmt.Sprintf("❌ <b>Appointment Canceled</b>\nYour session with %s on %s has been canceled by %s.",
				msg.Data["psychologist_name"], msg.Data["datetime"], msg.Data["cancelled_by"])
			err := telegram.SendMessage(tgChatID, tgText)
			if err != nil {
				log.Printf("Failed to send Telegram reminder: %v", err)
			}
		}

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
			err := telegram.SendMessage(tgChatID, tgText)
			if err != nil {
				log.Printf("Failed to send Telegram reminder: %v", err)
			}
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
