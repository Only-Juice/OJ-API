package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"

	"OJ-API/database"
	"OJ-API/models"
)

var (
	encryptionKey     string
	encryptionKeyOnce sync.Once
)

// getEncryptionKey returns the cached encryption key, initializing it if necessary
func getEncryptionKey() string {
	encryptionKeyOnce.Do(func() {
		encryptionKey = os.Getenv("ENCRYPTION_KEY")
	})
	return encryptionKey
}

// DecodeBase64Key decodes the Base64-encoded encryption key
func decodeBase64Key(encodedKey string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encodedKey)
}

// EncryptToken encrypts a token using AES-GCM
func EncryptToken(token, key string) (string, error) {
	decodedKey, err := decodeBase64Key(key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 12) // GCM nonce size is 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	cipherText := aesGCM.Seal(nil, nonce, []byte(token), nil)
	// Combine nonce and ciphertext for storage
	result := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptToken decrypts a token using AES-GCM
func DecryptToken(encryptedToken, key string) (string, error) {
	decodedKey, err := decodeBase64Key(key)
	if err != nil {
		return "", err
	}

	data, err := decodeBase64Key(encryptedToken)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return "", err
	}

	if len(data) < 12 {
		return "", errors.New("invalid encrypted token")
	}

	nonce := data[:12]
	cipherText := data[12:]

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

// StoreToken stores an encrypted token in the database
func StoreToken(userID uint, token string) error {
	encryptedToken, err := EncryptToken(token, getEncryptionKey())
	if err != nil {
		return err
	}

	db := database.DBConn
	return db.Model(&models.User{}).Where("id = ?", userID).Update("gitea_token", encryptedToken).Error
}

// GetToken retrieves and decrypts a token from the database
func GetToken(userID uint) (string, error) {
	var user models.User
	db := database.DBConn
	if err := db.Where("id = ?", userID).Limit(1).Find(&user).Error; err != nil {
		return "", err
	}

	return DecryptToken(user.GiteaToken, getEncryptionKey())
}
