package worker

import (
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
)

func StartTelegramBot() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Println("TELEGRAM_BOT_TOKEN not set. Telegram linking is disabled.")
		return
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Printf("Failed to connect to Telegram: %v", err)
		return
	}

	log.Printf("Authorized on Telegram account @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Listen for messages forever
	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			command := update.Message.Command()
			args := update.Message.CommandArguments()

			if command == "start" {
				chatID := update.Message.Chat.ID
				studentID := strings.TrimSpace(args)

				if studentID == "" {
					reply := tgbotapi.NewMessage(chatID, "Please provide your User ID. Format: /start <your-uuid>")
					bot.Send(reply)
					continue
				}

				var profile models.UserProfile
				if err := config.DB.First(&profile, "id = ?", studentID).Error; err != nil {
					reply := tgbotapi.NewMessage(chatID, "User not found. Please check your ID.")
					bot.Send(reply)
					continue
				}

				// Update their Telegram Chat ID
				chatIDStr := fmt.Sprintf("%d", chatID) // Convert int64 to string
				profile.TelegramChatID = chatIDStr

				if err := config.DB.Save(&profile).Error; err != nil {
					reply := tgbotapi.NewMessage(chatID, "Database error. Please try again later.")
					bot.Send(reply)
					continue
				}

				successMsg := fmt.Sprintf("Welcome, %s!\nYour Telegram account is now linked to KBTU Care. You will receive appointment reminders here.", profile.FullName)
				reply := tgbotapi.NewMessage(chatID, successMsg)
				bot.Send(reply)

				log.Printf("Linked Telegram Chat ID %s to User %s", chatIDStr, profile.ID)
			}
		}
	}
}
