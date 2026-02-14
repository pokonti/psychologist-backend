package utils

import (
	"math/rand"
	"time"
)

func GenerateRandomCode() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	digits := "0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = digits[rng.Intn(len(digits))]
	}
	return string(code)
}
