package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func SendMessage(chatID string, text string) error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" || chatID == "" {
		return fmt.Errorf("missing token or chat_id")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to send telegram message, status: %d", resp.StatusCode)
	}
	return nil
}
