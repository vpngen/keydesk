package auth

import (
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/keydesk/token"
	"net/http"
	"strings"
)

type Service struct {
	brigadeID string
}

func NewService(brigadeID string) Service {
	return Service{brigadeID: brigadeID}
}

func (s Service) Authorize(request *http.Request, i interface{}) error {
	return s.authorizeFunc(request, i.(authCtx))
}

func (s Service) BearerAuth(token string) (interface{}, error) {
	return s.authenticateBearerFunc(token)
}

type authCtx struct {
	claims  TokenClaims
	authReq *security.ScopedAuthRequest
}

func (s Service) authorizeFunc(_ *http.Request, authCtx authCtx) error {
	if authCtx.claims.Subject != s.brigadeID {
		return ErrUserUnknown
	}

	for _, scope := range authCtx.authReq.RequiredScopes {
		if !sliceContains(authCtx.claims.Scopes, scope) {
			return ErrMissingScopes
		}
	}

	return nil
}

func (s Service) authenticateBearerFunc(tokenStr string) (TokenClaims, error) {
	var claims TokenClaims
	jwtoken, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrTokenUnexpectedSigningMethod, t.Header["alg"])
		}
		return token.FetchSecret(claims.ID), nil
	})
	if err != nil {
		return TokenClaims{}, ErrTokenInvalid
	}

	if !jwtoken.Valid {
		return TokenClaims{}, ErrTokenInvalid
	}

	return claims, nil
}

func (s Service) APIKeyAuthenticator(_, _ string, authenticate security.TokenAuthentication) runtime.Authenticator {
	return runtime.AuthenticatorFunc(func(i interface{}) (bool, interface{}, error) {
		authReq := i.(*security.ScopedAuthRequest)
		claims, err := authenticate(strings.TrimPrefix(authReq.Request.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			return false, nil, err
		}
		return true, authCtx{claims: claims.(TokenClaims), authReq: authReq}, nil
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
