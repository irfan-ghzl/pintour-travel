package auth

import "github.com/golang-jwt/jwt/v5"

// Claims defines JWT payload fields shared between the token issuer (application layer)
// and the token verifier (delivery/middleware layer).
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
