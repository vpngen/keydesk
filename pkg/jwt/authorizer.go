package jwt

import (
	"crypto"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrTokenInvalid                 = errors.New("invalid token")
	ErrUserUnknown                  = errors.New("unknown user")
	ErrMissingScopes                = errors.New("missing scopes")
	ErrExternalIPMismatch           = errors.New("external IP mismatch")
	ErrInvalidVIP                   = errors.New("invalid VIP status")
)

type Authorizer struct {
	key     crypto.PublicKey
	options Options
}

func NewAuthorizer(key crypto.PublicKey, options Options) Authorizer {
	return Authorizer{key: key, options: options}
}

func (a Authorizer) Validate(tokenStr string) (Claims, error) {
	var claims Claims

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != a.options.SigningMethod.Alg() {
				return nil, ErrTokenUnexpectedSigningMethod
			}

			return a.key, nil
		},
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	if !token.Valid {
		return Claims{}, ErrTokenInvalid
	}

	return claims, nil
}

func (a Authorizer) Authorize(claims Claims, scopes ...string) error {
	if claims.Issuer != a.options.Issuer {
		return ErrMissingScopes
	}

	if claims.Subject != a.options.Subject {
		return ErrUserUnknown
	}

	for _, aud := range a.options.Audience {
		if !sliceContains(claims.Audience, aud) {
			return ErrMissingScopes
		}
	}

	for _, scope := range scopes {
		if !sliceContains(claims.Scopes, scope) {
			return ErrMissingScopes
		}
	}

	return nil
}

func sliceContains[T comparable](slice []T, val T) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

type KeydeskTokenAuthorizer struct {
	key     crypto.PublicKey
	options KeydeskTokenOptions
}

func NewKeydeskTokenAuthorizer(key crypto.PublicKey, options KeydeskTokenOptions) KeydeskTokenAuthorizer {
	return KeydeskTokenAuthorizer{key: key, options: options}
}

func (a KeydeskTokenAuthorizer) KeydeskTokenValidate(tokenStr string) (KeydeskTokenClaims, error) {
	var claims KeydeskTokenClaims
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != a.options.SigningMethod.Alg() {
				return nil, ErrTokenUnexpectedSigningMethod
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
	if a.options.Issuer != "" && claims.Issuer != a.options.Issuer {
		return ErrMissingScopes
	}

	if a.options.Subject != "" && claims.Subject != a.options.Subject {
		return ErrUserUnknown
	}

	if a.options.ExternalIP != "" && claims.ExternalIP != a.options.ExternalIP {
		return ErrMissingScopes
	}

	for _, aud := range a.options.Audience {
		if !sliceContains(claims.Audience, aud) {
			return ErrMissingScopes
		}
	}
	return nil
}
