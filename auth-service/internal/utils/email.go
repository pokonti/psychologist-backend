package utils

import (
	"errors"
	"fmt"
	"net/smtp"
	"os"
)

// loginAuth is a custom implementation of smtp.Auth for the LOGIN mechanism
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

var SendVerificationEmail = func(toEmail string, code string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if host == "" || port == "" || from == "" || password == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	auth := LoginAuth(from, password)

	subject := "Subject: Verify your account\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<html><body><h3>Your verification code is: <b>%s</b></h3></body></html>", code)

	headers := fmt.Sprintf("From: %s\nTo: %s\n", from, toEmail)
	msg := []byte(headers + subject + mime + body)

	// SendMail automatically handles STARTTLS if the port is 587
	err := smtp.SendMail(addr, auth, from, []string{toEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
