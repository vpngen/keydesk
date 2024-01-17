package auth

import "github.com/go-openapi/errors"

var (
	ErrTokenUnexpectedSigningMethod = errors.New(401, "unexpected signing method")
	ErrTokenInvalid                 = errors.New(401, "invalid token")
	ErrTokenExpired                 = errors.New(403, "token expired")
	ErrUserUnknown                  = errors.New(403, "unknown user")
	ErrMissingScopes                = errors.New(403, "missing scopes")
	ErrTokenCantSign                = "can't sign"
)
