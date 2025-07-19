package jwt

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	jwt.RegisteredClaims
	Scopes []string `json:"scopes"`
}

type KeydeskTokenClaims struct {
	jwt.RegisteredClaims
	Vip        bool   `json:"vip"`
	ExternalIP string `json:"external_ip,omitempty"`
}
