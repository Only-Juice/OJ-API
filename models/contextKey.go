package models

type contextKey string

const (
	JWTClaimsKey     contextKey = "jwtClaims"
	UserContextKey   contextKey = "user"
	ClientContextKey contextKey = "client"
)
