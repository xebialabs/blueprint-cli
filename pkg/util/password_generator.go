package util

import (
	"crypto/rand"
	"math/big"
)

const (
	lowercaseCharset = "abcdefghijklmnopqrstuvwxyz"
	uppercaseCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numericCharset   = "0123456789"
	completeCharset  = lowercaseCharset + uppercaseCharset + numericCharset
)

var charsets = [...]string{lowercaseCharset, uppercaseCharset, numericCharset}

func GeneratePassword(len int) string {
	password := ""
	for ; len > 3; len-- {
		password += randomElement(completeCharset)
	}
	for ; len > 0; len-- {
		password += randomElement(charsets[len-1])
	}
	return password
}

func randomElement(charset string) string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))

	if err != nil {
		Fatal("Error generating random password")
	}

	return string(charset[n.Int64()])
}
