package jwt

import (
	"crypto"
	"errors"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrTokenInvalid                 = errors.New("invalid token")
	ErrUserUnknown                  = errors.New("unknown user")
	ErrMissingScopes                = errors.New("missing scopes")
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
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != a.options.SigningMethod.Alg() {
				return nil, ErrTokenUnexpectedSigningMethod
			}
			return a.key, nil
		},
	)
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}
	if !token.Valid {
		return Claims{}, ErrTokenInvalid
	}
	return claims, nil
}

func (a Authorizer) Authorize(claims Claims, scopes ...string) error {
	if a.options.Issuer != "" && claims.Issuer != a.options.Issuer {
		return ErrMissingScopes
	}
	if a.options.Subject != "" && claims.Subject != a.options.Subject {
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
