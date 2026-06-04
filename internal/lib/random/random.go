package random

import (
	"crypto/rand"
	"encoding/base64"
)

func NewRandomString(length int) string {
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(b)[:length]
}
