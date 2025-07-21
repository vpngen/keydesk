package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenTooEarly = errors.New("token not valid yet")

	ErrTokenInvalid            = errors.New("invalid token")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrMissingAudiences        = errors.New("missing audiences")

	ErrUserUnknown = errors.New("unknown user")

	ErrMissingScopes = errors.New("missing scopes")
)

func checkTimeLimits(notBefore, expiresAt *jwt.NumericDate) error {
	now := time.Now().UTC()

	if notBefore != nil &&
		!notBefore.Time.IsZero() &&
		notBefore.Time.After(now) {
		return ErrTokenTooEarly
	}

	if expiresAt != nil &&
		!expiresAt.Time.IsZero() &&
		expiresAt.Time.Before(now) {
		return ErrTokenExpired
	}

	return nil
}
