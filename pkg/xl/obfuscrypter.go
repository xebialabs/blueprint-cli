package xl

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"bytes"
	"errors"
	"io"
	)

var fixedKey = []byte{
0x49, 0x37, 0x94, 0x8a, 0x1b, 0x32, 0x90, 0xb2, 0x5e, 0x46, 0x6e, 0x0c, 0x9b, 0x23, 0x86, 0xc6,
0xf9, 0xf3, 0x17, 0xcc, 0x8e, 0x44, 0x2d, 0x61, 0xce, 0xdb, 0x4e, 0x23, 0xae, 0xc5, 0x6e, 0xa5}

func pad(unpaddedBytes []byte) []byte {
	paddingLength := aes.BlockSize - len(unpaddedBytes)%aes.BlockSize
	padBytes := bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)
	return append(unpaddedBytes, padBytes...)
}

func unpad(paddedBytes []byte) ([]byte, error) {
	length := len(paddedBytes)
	paddingLength := int(paddedBytes[length-1])

	if paddingLength > length {
		return nil, errors.New("unpad error")
	}

	return paddedBytes[:(length - paddingLength)], nil
}

func Obfuscrypt(text string) (string, error) {
	block, err := aes.NewCipher(fixedKey)
	if err != nil {
		return "", err
	}

	paddedBytes := pad([]byte(text))
	encryptedBytes := make([]byte, aes.BlockSize + len(paddedBytes))
	iv := encryptedBytes[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(encryptedBytes[aes.BlockSize:], paddedBytes)
	obfuscryptedText := base64.StdEncoding.EncodeToString(encryptedBytes)
	return obfuscryptedText, nil
}

func Deobfuscrypt(obfuscryptedText string) (string, error) {
	block, err := aes.NewCipher(fixedKey)
	if err != nil {
		return "", err
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(obfuscryptedText)
	if err != nil {
		return "", err
	}

	if (len(encryptedBytes) == 0) {
		return "", errors.New("Blocksize must be greater than 0")
	}

	if (len(encryptedBytes) % aes.BlockSize) != 0 {
		return "", errors.New("Blocksize must be multipe of decoded message length")
	}

	iv := encryptedBytes[:aes.BlockSize]
	encryptedPaddedBytes := encryptedBytes[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(encryptedPaddedBytes, encryptedPaddedBytes)

	deobfuscryptedText, err := unpad(encryptedPaddedBytes)
	if err != nil {
		return "", err
	}
	return string(deobfuscryptedText), nil
}
