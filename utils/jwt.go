package utils

import (
	"OJ-API/config"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret     string
	jwtSecretOnce sync.Once
)

// getJWTSecret returns the cached JWT secret, initializing it if necessary
func getJWTSecret() string {
	jwtSecretOnce.Do(func() {
		jwtSecret = config.Config("JWT_SECRET")
	})
	return jwtSecret
}

type JWTClaims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	IsAdmin   bool   `json:"is_admin"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a short-lived access token (15 minutes)
func GenerateAccessToken(userID uint, username string, isAdmin bool) (string, error) {
	claims := JWTClaims{
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// GenerateRefreshToken generates a long-lived refresh token (7 days)
func GenerateRefreshToken(userID uint, username string, isAdmin bool) (string, error) {
	claims := JWTClaims{
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// GenerateTokens generates both access and refresh tokens
func GenerateTokens(userID uint, username string, isAdmin bool) (accessToken, refreshToken string, err error) {
	accessToken, err = GenerateAccessToken(userID, username, isAdmin)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = GenerateRefreshToken(userID, username, isAdmin)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateJWT generates a JWT token (deprecated, use GenerateAccessToken instead)
func GenerateJWT(userID uint, username string, isAdmin bool) (string, error) {
	return GenerateAccessToken(userID, username, isAdmin)
}

// ParseJWT parses a JWT token
func ParseJWT(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(getJWTSecret()), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns claims if valid
func ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	claims, err := ParseJWT(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// ValidateAccessToken validates an access token and returns claims if valid
func ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	claims, err := ParseJWT(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
