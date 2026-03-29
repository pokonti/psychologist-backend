package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ResendPayload is the structure required by the Resend API
type ResendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

func SendEmail(toEmail string, subject string, bodyHTML string) error {
	apiKey := os.Getenv("RESEND_API_KEY")

	from := "KBTU Care <notifications@kbtucare.site>"

	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is missing")
	}

	payload := ResendPayload{
		From:    from,
		To:      []string{toEmail},
		Subject: subject,
		Html:    bodyHTML,
	}

	jsonData, _ := json.Marshal(payload)

	// Send request via HTTPS (Port 443)
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Resend API error: status %d", resp.StatusCode)
	}

	return nil
}
