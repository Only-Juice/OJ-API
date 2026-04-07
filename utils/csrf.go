package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

const (
	csrfTokenExpiration = 24 * time.Hour
	randomBytesLength   = 16 // Length of random bytes
)

// GenerateCSRFToken generates a new CSRF token with a timestamp and random bytes
func GenerateCSRFToken() (string, error) {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, randomBytesLength)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	plaintext := fmt.Sprintf("%x|%d", randomBytes, timestamp)

	return EncryptToken(plaintext, getEncryptionKey())
}

// ValidateCSRFToken validates a CSRF token by decrypting and checking the timestamp
func ValidateCSRFToken(token string) error {
	plaintext, err := DecryptToken(token, getEncryptionKey())
	if err != nil {
		return errors.New("invalid CSRF token")
	}

	parts := string(plaintext)
	var randomPart string
	var timestamp int64
	_, err = fmt.Sscanf(parts, "%x|%d", &randomPart, &timestamp)
	if err != nil {
		return errors.New("invalid CSRF token")
	}

	if time.Since(time.Unix(timestamp, 0)) > csrfTokenExpiration {
		return errors.New("CSRF token has expired")
	}

	return nil
}
