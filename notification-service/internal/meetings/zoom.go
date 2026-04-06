package meetings

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type ZoomClient struct {
	AccountID    string
	ClientID     string
	ClientSecret string
}

func NewZoomClient() *ZoomClient {
	return &ZoomClient{
		AccountID:    os.Getenv("ZOOM_ACCOUNT_ID"),
		ClientID:     os.Getenv("ZOOM_CLIENT_ID"),
		ClientSecret: os.Getenv("ZOOM_CLIENT_SECRET"),
	}
}

// GetAccessToken gets the token using Server-to-Server OAuth
func (z *ZoomClient) GetAccessToken() (string, error) {
	url := fmt.Sprintf("https://zoom.us/oauth/token?grant_type=account_credentials&account_id=%s", z.AccountID)

	auth := base64.StdEncoding.EncodeToString([]byte(z.ClientID + ":" + z.ClientSecret))

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	token, ok := res["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get zoom token: %v", res)
	}
	return token, nil
}

// CreateMeeting generates a Zoom link
func (z *ZoomClient) CreateMeeting(topic string, startTime time.Time) (string, error) {
	token, err := z.GetAccessToken()
	if err != nil {
		return "", err
	}

	// 'me' refers to the admin account that owns the app
	url := "https://api.zoom.us/v2/users/me/meetings"

	body := map[string]interface{}{
		"topic":      topic,
		"type":       2, // Scheduled meeting
		"start_time": startTime.Format("2006-01-02T15:04:05"),
		"duration":   50,
		"timezone":   "Asia/Almaty",
		"settings": map[string]interface{}{
			"join_before_host": true,
			"jbh_time":         0,
			"waiting_room":     false,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	joinURL, ok := result["join_url"].(string)
	if !ok {
		return "", fmt.Errorf("zoom error: %v", result)
	}

	return joinURL, nil
}
