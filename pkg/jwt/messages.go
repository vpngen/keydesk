package jwt

import (
	"crypto"
	"fmt"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type MessagesJwtOptions struct {
	// if Issuer is empty, all issuers are allowed
	Issuer string
	// if Subject is empty, all subjects are allowed
	Subject string
	// if Audience is empty, all audiences are allowed
	Audience []string

	SigningMethod jwt.SigningMethod
}

type MessagesJwtIssuer struct {
	key     crypto.PrivateKey
	options MessagesJwtOptions
}

type MessagesJwtClaims struct {
	jwt.RegisteredClaims

	Scopes []string `json:"scopes"`
}

func NewMessagesJwtIssuer(key crypto.PrivateKey, options MessagesJwtOptions) MessagesJwtIssuer {
	return MessagesJwtIssuer{key: key, options: options}
}

func (i MessagesJwtIssuer) IsNil() bool {
	return i.key == nil
}

func (i MessagesJwtIssuer) CreateToken(ttl time.Duration, scopes ...string) MessagesJwtClaims {
	now := time.Now()
	return MessagesJwtClaims{
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

func (i MessagesJwtIssuer) Sign(claims MessagesJwtClaims) (string, error) {
	return jwt.NewWithClaims(i.options.SigningMethod, claims).SignedString(i.key)
}

type MessagesJwtAuthorizer struct {
	key     crypto.PublicKey
	options MessagesJwtOptions
}

func NewMessagesJwtAuthorizer(key crypto.PublicKey, options MessagesJwtOptions) MessagesJwtAuthorizer {
	return MessagesJwtAuthorizer{key: key, options: options}
}

func (a MessagesJwtAuthorizer) IsNil() bool {
	return a.key == nil
}

func (a MessagesJwtAuthorizer) Validate(tokenStr string) (MessagesJwtClaims, error) {
	var claims MessagesJwtClaims

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
		return MessagesJwtClaims{}, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	if !token.Valid {
		return MessagesJwtClaims{}, ErrTokenInvalid
	}

	return claims, nil
}

func (a MessagesJwtAuthorizer) Authorize(claims MessagesJwtClaims, scopes ...string) error {
	if err := checkTimeLimits(claims.NotBefore, claims.ExpiresAt); err != nil {
		return err
	}

	if claims.Issuer != a.options.Issuer {
		return ErrMissingScopes
	}

	if claims.Subject != a.options.Subject {
		return ErrUserUnknown
	}

	for _, aud := range a.options.Audience {
		if !slices.Contains(claims.Audience, aud) {
			return ErrMissingAudiences
		}
	}

	for _, scope := range scopes {
		if !slices.Contains(claims.Scopes, scope) {
			return ErrMissingScopes
		}
	}

	return nil
}
