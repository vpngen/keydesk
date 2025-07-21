package jwt

import (
	"crypto"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type KeydeskTokenOptions struct {
	// if Issuer is empty, all issuers are allowed
	Issuer string
	// if Subject is empty, all subjects are allowed
	Subject string
	// if Audience is empty, all audiences are allowed
	Audience []string

	// My IP address, used to verify the token
	ExternalIP string

	SigningMethod jwt.SigningMethod
}

type KeydeskTokenIssuer struct {
	key   crypto.PrivateKey
	keyId string

	options KeydeskTokenOptions
}

type KeydeskTokenClaims struct {
	jwt.RegisteredClaims
	Vip        bool   `json:"vip"`
	ExternalIP string `json:"external_ip,omitempty"`
}

var (
	ErrExternalIPMismatch = errors.New("external IP mismatch")
	ErrInvalidVIP         = errors.New("invalid VIP status")
)

func NewKeydeskTokenIssuer(key crypto.PrivateKey, keyId string, options KeydeskTokenOptions) KeydeskTokenIssuer {
	return KeydeskTokenIssuer{
		key:   key,
		keyId: keyId,

		options: options,
	}
}

func (i KeydeskTokenIssuer) IsNil() bool {
	return i.key == nil
}

func (i KeydeskTokenIssuer) CreateToken(ttl time.Duration, vip bool) KeydeskTokenClaims {
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

func (i KeydeskTokenIssuer) Sign(claims KeydeskTokenClaims) (string, error) {
	token := jwt.NewWithClaims(i.options.SigningMethod, claims)
	token.Header["kid"] = i.keyId

	return token.SignedString(i.key)
}

type KeydeskTokenAuthorizer struct {
	key     crypto.PublicKey
	options KeydeskTokenOptions
}

func NewKeydeskTokenAuthorizer(key crypto.PublicKey, options KeydeskTokenOptions) KeydeskTokenAuthorizer {
	return KeydeskTokenAuthorizer{key: key, options: options}
}

func (a KeydeskTokenAuthorizer) IsNil() bool {
	return a.key == nil
}

func (a KeydeskTokenAuthorizer) Validate(tokenStr string) (KeydeskTokenClaims, error) {
	var claims KeydeskTokenClaims
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != a.options.SigningMethod.Alg() {
				return nil, ErrUnexpectedSigningMethod
			}

			return a.key, nil
		},
	)
	if err != nil {
		return KeydeskTokenClaims{}, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	if !token.Valid {
		return KeydeskTokenClaims{}, ErrTokenInvalid
	}

	return claims, nil
}

func (a KeydeskTokenAuthorizer) Authorize(claims KeydeskTokenClaims, externalIP string, vip bool) error {
	if err := checkTimeLimits(claims.NotBefore, claims.ExpiresAt); err != nil {
		return err
	}

	if claims.Issuer != a.options.Issuer {
		return ErrMissingScopes
	}

	if claims.Subject != a.options.Subject {
		return ErrUserUnknown
	}

	if claims.ExternalIP != a.options.ExternalIP {
		return ErrMissingScopes
	}

	for _, aud := range a.options.Audience {
		if !slices.Contains(claims.Audience, aud) {
			return ErrMissingScopes
		}
	}
	return nil
}
