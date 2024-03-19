package util

import (
	"crypto/rand"
	"math/big"
	"unicode"
)

const (
	PasswordChar = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	Length       = len(PasswordChar)
)

func GeneratePassword(len int) string {
	password := ""

	if len > 0 {
		if len > 1 {
			for ok := true; ok; ok = !hasNumeric(password) {
				password = ""

				for i := 1; i <= len; i++ {
					password += randomElement()
				}
			}
		} else {
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

func hasNumeric(s string) bool {
	for _, r := range s {
		if unicode.IsNumber(r) {
			return true
		}
	}
	return false
}
