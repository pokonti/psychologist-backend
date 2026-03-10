package email

import (
	"errors"
	"fmt"
	"net/smtp"
	"os"
)

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}
func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}
func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown from server")
		}
	}
	return nil, nil
}

func SendEmail(toEmail string, subject string, bodyHTML string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if host == "" || port == "" || from == "" || password == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	auth := LoginAuth(from, password)

	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	headers := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n", from, toEmail, subject)
	msg := []byte(headers + mime + bodyHTML)

	if err := smtp.SendMail(addr, auth, from, []string{toEmail}, msg); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
