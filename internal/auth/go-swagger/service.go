package go_swagger

import (
	errors2 "errors"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/security"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/pkg/jwt"
	"net/http"
	"strings"
)

type Service struct {
	authorizer jwt.Authorizer
}

func NewService(authorizer jwt.Authorizer) Service {
	return Service{authorizer: authorizer}
}

func (s Service) Authorize(request *http.Request, i interface{}) error {
	return s.authorizeFunc(request, i.(authCtx))
}

func (s Service) BearerAuth(token string) (interface{}, error) {
	return s.authenticateBearerFunc(token)
}

type authCtx struct {
	claims  jwt.Claims
	authReq *security.ScopedAuthRequest
}

func (s Service) authorizeFunc(_ *http.Request, authCtx authCtx) error {
	if err := s.authorizer.Authorize(authCtx.claims, authCtx.authReq.RequiredScopes...); err != nil {
		return wrapError(err)
	}
	return nil
}

func (s Service) authenticateBearerFunc(tokenStr string) (jwt.Claims, error) {
	claims, err := s.authorizer.Validate(tokenStr)
	if err != nil {
		return jwt.Claims{}, wrapError(err)
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
		return true, authCtx{claims: claims.(jwt.Claims), authReq: authReq}, nil
	})
}

var (
	ErrTokenExpired                 = errors.New(403, "token expired")
	ErrTokenCantSign                = "can't sign"
	ErrTokenUnexpectedSigningMethod = errors.New(401, "unexpected signing method")
	ErrTokenInvalid                 = errors.New(401, "invalid token")
	ErrUserUnknown                  = errors.New(403, "unknown user")
	ErrMissingScopes                = errors.New(403, "missing scopes")
)

func wrapError(err error) error {
	if errors2.Is(err, jwt2.ErrTokenSignatureInvalid) {
		return ErrTokenUnexpectedSigningMethod
	}
	if errors2.Is(err, jwt.ErrTokenInvalid) {
		return ErrTokenInvalid
	}
	if errors2.Is(err, jwt.ErrUserUnknown) {
		return ErrUserUnknown
	}
	if errors2.Is(err, jwt.ErrMissingScopes) {
		return ErrMissingScopes
	}
	return err
}
