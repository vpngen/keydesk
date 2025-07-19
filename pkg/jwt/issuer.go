package jwt

import (
	"crypto"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Issuer struct {
	key     crypto.PrivateKey
	options Options
}

func NewIssuer(key crypto.PrivateKey, options Options) Issuer {
	return Issuer{key: key, options: options}
}

func (i Issuer) CreateToken(ttl time.Duration, scopes ...string) Claims {
	now := time.Now()
	return Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.options.Issuer,
			Subject:   i.options.Subject,
			Audience:  i.options.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		Scopes: scopes,
	}
}

func (i Issuer) Sign(claims Claims) (string, error) {
	return jwt.NewWithClaims(i.options.SigningMethod, claims).SignedString(i.key)
}

type KeydeskTokenIssuer struct {
	key     crypto.PrivateKey
	options KeydeskTokenOptions
}

func NewKeydeskTokenIssuer(key crypto.PrivateKey, options KeydeskTokenOptions) KeydeskTokenIssuer {
	return KeydeskTokenIssuer{
		key:     key,
		options: options,
	}
}

func (i KeydeskTokenIssuer) CreateKeydeskToken(ttl time.Duration, vip bool) KeydeskTokenClaims {
	now := time.Now()
	return KeydeskTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.options.Issuer,
			Subject:   i.options.Subject,
			Audience:  i.options.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		ExternalIP: i.options.ExternalIP,
		Vip:        vip,
	}
}

func (i KeydeskTokenIssuer) SignKeydeskToken(claims KeydeskTokenClaims) (string, error) {
	return jwt.NewWithClaims(i.options.SigningMethod, claims).SignedString(i.key)
}
