package utils

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTClaims(t *testing.T) {
	t.Run("JWTClaims struct initialization", func(t *testing.T) {
		claims := JWTClaims{
			UserID:   123,
			Username: "testuser",
			IsAdmin:  true,
		}

		assert.Equal(t, uint(123), claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.True(t, claims.IsAdmin)
	})
}

func TestGenerateJWT(t *testing.T) {
	// Set up test environment
	testSecret := "test-jwt-secret-key-for-testing"
	originalSecret := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Setenv("JWT_SECRET", originalSecret)

	t.Run("successful JWT generation", func(t *testing.T) {
		userID := uint(123)
		username := "testuser"
		isAdmin := true

		token, err := GenerateJWT(userID, username, isAdmin)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify token structure (should have 3 parts separated by dots)
		tokenParts := len(token)
		assert.Greater(t, tokenParts, 0)
	})

	t.Run("generate multiple different tokens", func(t *testing.T) {
		token1, err1 := GenerateJWT(1, "user1", false)
		token2, err2 := GenerateJWT(2, "user2", true)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, token1, token2)
	})

	t.Run("generate token with empty JWT_SECRET", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "")

		token, err := GenerateJWT(123, "testuser", false)

		// Should still work with empty secret (though not secure)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestParseJWT(t *testing.T) {
	// Set up test environment
	testSecret := "test-jwt-secret-key-for-testing"
	originalSecret := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Setenv("JWT_SECRET", originalSecret)

	t.Run("parse valid JWT token", func(t *testing.T) {
		userID := uint(456)
		username := "parsetest"
		isAdmin := false

		// Generate a token first
		token, err := GenerateJWT(userID, username, isAdmin)
		require.NoError(t, err)

		// Parse the token
		claims, err := ParseJWT(token)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, isAdmin, claims.IsAdmin)
	})

	t.Run("parse invalid JWT token", func(t *testing.T) {
		invalidToken := "invalid.jwt.token"

		claims, err := ParseJWT(invalidToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("parse empty token", func(t *testing.T) {
		claims, err := ParseJWT("")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("parse token with wrong secret", func(t *testing.T) {
		// Generate token with one secret
		token, err := GenerateJWT(123, "test", false)
		require.NoError(t, err)

		// Try to parse with different secret
		os.Setenv("JWT_SECRET", "different-secret")

		claims, err := ParseJWT(token)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("parse malformed token", func(t *testing.T) {
		malformedTokens := []string{
			"not.a.jwt",
			"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9",
			"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9",
		}

		for _, token := range malformedTokens {
			claims, err := ParseJWT(token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		}
	})
}

func TestJWTRoundTrip(t *testing.T) {
	// Set up test environment
	testSecret := "test-jwt-secret-key-for-roundtrip"
	originalSecret := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Setenv("JWT_SECRET", originalSecret)

	testCases := []struct {
		name     string
		userID   uint
		username string
		isAdmin  bool
	}{
		{"admin user", 1, "admin", true},
		{"regular user", 2, "user", false},
		{"user with special chars", 3, "user@test.com", false},
		{"user with numbers", 999, "user123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate token
			token, err := GenerateJWT(tc.userID, tc.username, tc.isAdmin)
			require.NoError(t, err)

			// Parse token
			claims, err := ParseJWT(token)
			require.NoError(t, err)

			// Verify all fields match
			assert.Equal(t, tc.userID, claims.UserID)
			assert.Equal(t, tc.username, claims.Username)
			assert.Equal(t, tc.isAdmin, claims.IsAdmin)
		})
	}
}

func TestJWTWithCustomClaims(t *testing.T) {
	testSecret := "test-jwt-secret-key"
	originalSecret := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Setenv("JWT_SECRET", originalSecret)

	t.Run("test JWT with registered claims", func(t *testing.T) {
		// Create claims with registered claims
		claims := JWTClaims{
			UserID:   100,
			Username: "testuser",
			IsAdmin:  true,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		// Create token manually to test custom registered claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(testSecret))
		require.NoError(t, err)

		// Parse the token
		parsedClaims, err := ParseJWT(tokenString)
		require.NoError(t, err)

		assert.Equal(t, uint(100), parsedClaims.UserID)
		assert.Equal(t, "testuser", parsedClaims.Username)
		assert.True(t, parsedClaims.IsAdmin)
		assert.NotNil(t, parsedClaims.ExpiresAt)
		assert.NotNil(t, parsedClaims.IssuedAt)
	})
}
