package util

import (
	"crypto/rand"
	"math/big"
)

const (
	PasswordChar = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	Length       = len(PasswordChar)
)

func GeneratePassword(len int) string {
	password := ""
	if len > 0 {
		for i := 1; i <= len; i++ {
			password += randomElement()
		}
	}
	return password
}

func randomElement() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(Length)))

	if err != nil {
		Fatal("Error generating random password")
	}

	return string(PasswordChar[n.Int64()])
}
