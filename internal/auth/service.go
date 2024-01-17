package auth

import (
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/keydesk/token"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	// if Issuer is empty, all issuers are allowed
	Issuer string
	// if Subject is empty, all subjects are allowed
	Subject string
	// if Audience is empty, all audiences are allowed
	Audience []string
}

func (s Service) NewClaims(scopes []string, exp time.Time, id string) Claims {
	now := time.Now()
	return Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.Issuer,
			Subject:   s.Subject,
			Audience:  s.Audience,
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        id,
		},
		Scopes: scopes,
	}
}

func (s Service) Authorize(request *http.Request, i interface{}) error {
	return s.authorizeFunc(request, i.(authCtx))
}

func (s Service) BearerAuth(token string) (interface{}, error) {
	return s.authenticateBearerFunc(token)
}

type authCtx struct {
	claims  Claims
	authReq *security.ScopedAuthRequest
}

func (s Service) authorizeFunc(_ *http.Request, authCtx authCtx) error {
	if s.Issuer != "" && authCtx.claims.Issuer != s.Issuer {
		return ErrMissingScopes
	}

	if s.Subject != "" && authCtx.claims.Subject != s.Subject {
		return ErrUserUnknown
	}

	for _, aud := range s.Audience {
		if !sliceContains(authCtx.claims.Audience, aud) {
			return ErrMissingScopes
		}
	}

	for _, scope := range authCtx.authReq.RequiredScopes {
		if !sliceContains(authCtx.claims.Scopes, scope) {
			return ErrMissingScopes
		}
	}

	return nil
}

func (s Service) authenticateBearerFunc(tokenStr string) (Claims, error) {
	var claims Claims
	jwtoken, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrTokenUnexpectedSigningMethod, t.Header["alg"])
		}
		return token.FetchSecret(claims.ID), nil
	})
	if err != nil {
		return Claims{}, ErrTokenInvalid
	}

	if !jwtoken.Valid {
		return Claims{}, ErrTokenInvalid
	}

	return claims, nil
}

func (s Service) APIKeyAuthenticator(name, _ string, authenticate security.TokenAuthentication) runtime.Authenticator {
	return runtime.AuthenticatorFunc(func(i interface{}) (bool, interface{}, error) {
		authReq := i.(*security.ScopedAuthRequest)
		claims, err := authenticate(strings.TrimPrefix(authReq.Request.Header.Get(name), "Bearer "))
		if err != nil {
			return false, nil, err
		}
		return true, authCtx{claims: claims.(Claims), authReq: authReq}, nil
	})
}

func sliceContains[T comparable](slice []T, val T) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
