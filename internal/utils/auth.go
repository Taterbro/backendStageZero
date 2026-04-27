package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
