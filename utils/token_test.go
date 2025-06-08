package utils

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to generate a valid base64 encoded AES key
func generateTestKey() string {
	key := make([]byte, 32) // 256-bit key for AES-256
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}

func TestDecodeBase64Key(t *testing.T) {
	t.Run("decode valid base64 key", func(t *testing.T) {
		originalData := []byte("this is a test key 32 bytes long")
		encodedKey := base64.StdEncoding.EncodeToString(originalData)

		decodedKey, err := decodeBase64Key(encodedKey)

		assert.NoError(t, err)
		assert.Equal(t, originalData, decodedKey)
	})

	t.Run("decode invalid base64 key", func(t *testing.T) {
		invalidKey := "invalid-base64-key!"

		decodedKey, err := decodeBase64Key(invalidKey)

		assert.Error(t, err)
		assert.NotNil(t, decodedKey) // Go's base64 decoder is lenient and may return partial data
	})

	t.Run("decode empty key", func(t *testing.T) {
		decodedKey, err := decodeBase64Key("")

		assert.NoError(t, err)
		assert.Empty(t, decodedKey)
	})
}

func TestEncryptToken(t *testing.T) {
	testKey := generateTestKey()

	t.Run("encrypt valid token", func(t *testing.T) {
		token := "test-token-to-encrypt"

		encryptedToken, err := EncryptToken(token, testKey)

		assert.NoError(t, err)
		assert.NotEmpty(t, encryptedToken)
		assert.NotEqual(t, token, encryptedToken)

		// Verify it's base64 encoded
		_, err = base64.StdEncoding.DecodeString(encryptedToken)
		assert.NoError(t, err)
	})

	t.Run("encrypt empty token", func(t *testing.T) {
		encryptedToken, err := EncryptToken("", testKey)

		assert.NoError(t, err)
		assert.NotEmpty(t, encryptedToken)
	})

	t.Run("encrypt with invalid key", func(t *testing.T) {
		invalidKey := "invalid-key"
		token := "test-token"

		encryptedToken, err := EncryptToken(token, invalidKey)

		assert.Error(t, err)
		assert.Empty(t, encryptedToken)
	})

	t.Run("encrypt with wrong key length", func(t *testing.T) {
		// Create a base64 encoded key that's too short for AES
		shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
		token := "test-token"

		encryptedToken, err := EncryptToken(token, shortKey)

		assert.Error(t, err)
		assert.Empty(t, encryptedToken)
	})

	t.Run("encrypt different tokens produce different results", func(t *testing.T) {
		token1 := "token1"
		token2 := "token2"

		encrypted1, err1 := EncryptToken(token1, testKey)
		encrypted2, err2 := EncryptToken(token2, testKey)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, encrypted1, encrypted2)
	})

	t.Run("encrypt same token multiple times produces different results", func(t *testing.T) {
		token := "same-token"

		encrypted1, err1 := EncryptToken(token, testKey)
		encrypted2, err2 := EncryptToken(token, testKey)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		// Should be different due to random nonce
		assert.NotEqual(t, encrypted1, encrypted2)
	})
}

func TestDecryptToken(t *testing.T) {
	testKey := generateTestKey()

	t.Run("decrypt valid encrypted token", func(t *testing.T) {
		originalToken := "test-token-to-decrypt"

		encryptedToken, err := EncryptToken(originalToken, testKey)
		require.NoError(t, err)

		decryptedToken, err := DecryptToken(encryptedToken, testKey)

		assert.NoError(t, err)
		assert.Equal(t, originalToken, decryptedToken)
	})

	t.Run("decrypt with wrong key", func(t *testing.T) {
		originalToken := "test-token"
		wrongKey := generateTestKey()

		encryptedToken, err := EncryptToken(originalToken, testKey)
		require.NoError(t, err)

		decryptedToken, err := DecryptToken(encryptedToken, wrongKey)

		assert.Error(t, err)
		assert.Empty(t, decryptedToken)
	})

	t.Run("decrypt invalid encrypted token", func(t *testing.T) {
		invalidEncryptedToken := "invalid-encrypted-token"

		decryptedToken, err := DecryptToken(invalidEncryptedToken, testKey)

		assert.Error(t, err)
		assert.Empty(t, decryptedToken)
	})

	t.Run("decrypt empty encrypted token", func(t *testing.T) {
		decryptedToken, err := DecryptToken("", testKey)

		assert.Error(t, err)
		assert.Empty(t, decryptedToken)
	})

	t.Run("decrypt truncated encrypted token", func(t *testing.T) {
		originalToken := "test-token"
		encryptedToken, err := EncryptToken(originalToken, testKey)
		require.NoError(t, err)

		// Truncate the encrypted token to make it invalid
		truncatedToken := encryptedToken[:10]

		decryptedToken, err := DecryptToken(truncatedToken, testKey)

		assert.Error(t, err)
		assert.Empty(t, decryptedToken)
	})

	t.Run("decrypt with invalid key format", func(t *testing.T) {
		originalToken := "test-token"
		encryptedToken, err := EncryptToken(originalToken, testKey)
		require.NoError(t, err)

		invalidKey := "not-base64-key!"

		decryptedToken, err := DecryptToken(encryptedToken, invalidKey)

		assert.Error(t, err)
		assert.Empty(t, decryptedToken)
	})
}

func TestTokenEncryptionRoundTrip(t *testing.T) {
	testKey := generateTestKey()

	testCases := []string{
		"simple-token",
		"token-with-special-chars!@#$%^&*()",
		"very-long-token-that-contains-a-lot-of-text-to-test-encryption-with-longer-content",
		"",
		"123456789",
		"token with spaces",
		"token\nwith\nnewlines",
		"token\twith\ttabs",
		"token-with-unicode-🚀-chars",
	}

	for _, token := range testCases {
		t.Run("round trip for: "+token, func(t *testing.T) {
			// Encrypt
			encryptedToken, err := EncryptToken(token, testKey)
			require.NoError(t, err)

			// Decrypt
			decryptedToken, err := DecryptToken(encryptedToken, testKey)
			require.NoError(t, err)

			// Verify
			assert.Equal(t, token, decryptedToken)
		})
	}
}

func TestStoreToken(t *testing.T) {
	t.Run("store token function exists", func(t *testing.T) {
		// Test that the function exists and has the correct signature
		// We can't test the actual database operations without a database connection
		// but we can verify the function is available
		assert.NotNil(t, StoreToken)
	})
}

func TestGetToken(t *testing.T) {
	t.Run("get token function exists", func(t *testing.T) {
		// Test that the function exists and has the correct signature
		// We can't test the actual database operations without a database connection
		// but we can verify the function is available
		assert.NotNil(t, GetToken)
	})
}

func TestTokenFunctionsWithEnvironmentVariables(t *testing.T) {
	t.Run("test functions with missing ENCRYPTION_KEY", func(t *testing.T) {
		originalKey := os.Getenv("ENCRYPTION_KEY")
		os.Unsetenv("ENCRYPTION_KEY")
		defer os.Setenv("ENCRYPTION_KEY", originalKey)

		token := "test-token"

		// EncryptToken should fail with empty key
		encrypted, err := EncryptToken(token, os.Getenv("ENCRYPTION_KEY"))
		assert.Error(t, err)
		assert.Empty(t, encrypted)

		// DecryptToken should fail with empty key
		decrypted, err := DecryptToken("some-encrypted-data", os.Getenv("ENCRYPTION_KEY"))
		assert.Error(t, err)
		assert.Empty(t, decrypted)
	})

	t.Run("test with empty ENCRYPTION_KEY", func(t *testing.T) {
		originalKey := os.Getenv("ENCRYPTION_KEY")
		os.Setenv("ENCRYPTION_KEY", "")
		defer os.Setenv("ENCRYPTION_KEY", originalKey)

		token := "test-token"

		// Should fail with empty key
		encrypted, err := EncryptToken(token, os.Getenv("ENCRYPTION_KEY"))
		assert.Error(t, err)
		assert.Empty(t, encrypted)
	})
}

func TestEdgeCases(t *testing.T) {
	testKey := generateTestKey()

	t.Run("encrypt and decrypt very large token", func(t *testing.T) {
		// Create a large token (1MB)
		largeToken := make([]byte, 1024*1024)
		for i := range largeToken {
			largeToken[i] = byte(i % 256)
		}
		tokenStr := string(largeToken)

		encrypted, err := EncryptToken(tokenStr, testKey)
		require.NoError(t, err)

		decrypted, err := DecryptToken(encrypted, testKey)
		require.NoError(t, err)

		assert.Equal(t, tokenStr, decrypted)
	})

	t.Run("test with binary data", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		tokenStr := string(binaryData)

		encrypted, err := EncryptToken(tokenStr, testKey)
		require.NoError(t, err)

		decrypted, err := DecryptToken(encrypted, testKey)
		require.NoError(t, err)

		assert.Equal(t, tokenStr, decrypted)
	})
}
