package middleware

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(getSecret())

func getSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "supersecret"
	}
	return secret
}

func GenerateJWT(userID, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID, // used by gateway as X-User-ID
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
