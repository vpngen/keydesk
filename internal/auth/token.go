package auth

import "github.com/golang-jwt/jwt/v5"

type TokenClaims struct {
	jwt.RegisteredClaims
	Scopes []string `json:"scopes"`
}
