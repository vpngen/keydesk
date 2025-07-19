package go_swagger

import (
	errors2 "errors"

	"github.com/go-openapi/errors"
	"github.com/golang-jwt/jwt/v5"
	jwtsvc "github.com/vpngen/keydesk/pkg/jwt"
)

type Service struct {
	authorizer jwtsvc.KeydeskTokenAuthorizer
}

func NewService(authorizer jwtsvc.KeydeskTokenAuthorizer) Service {
	return Service{authorizer: authorizer}
}

/*func (s Service) Authorize(request *http.Request, i any) error {
	return s.authorizeFunc(request, i.(authCtx))
}*/

func (s Service) BearerAuth(token string) (any, error) {
	return s.authenticateBearerFunc(token)
}

/*type authCtx struct {
	claims  jwtsvc.KeydeskTokenClaims
	authReq *security.ScopedAuthRequest
}*/

/*func (s Service) authorizeFunc(_ *http.Request, authCtx authCtx) error {
	if err := s.authorizer.Authorize(authCtx.claims); err != nil {
		return wrapError(err)
	}

	return nil
}*/

func (s Service) authenticateBearerFunc(tokenStr string) (jwtsvc.KeydeskTokenClaims, error) {
	claims, err := s.authorizer.KeydeskTokenValidate(tokenStr)
	if err != nil {
		return jwtsvc.KeydeskTokenClaims{}, wrapError(err)
	}

	return claims, nil
}

/*func (s Service) APIKeyAuthenticator(name, _ string, authenticate security.TokenAuthentication) runtime.Authenticator {
	return runtime.AuthenticatorFunc(func(i interface{}) (bool, interface{}, error) {
		authReq := i.(*security.ScopedAuthRequest)
		claims, err := authenticate(strings.TrimPrefix(authReq.Request.Header.Get(name), "Bearer "))
		if err != nil {
			return false, nil, err
		}
		return true, authCtx{claims: claims.(jwtsvc.Claims), authReq: authReq}, nil
	})
}*/

var (
	ErrTokenExpired                 = errors.New(403, "token expired")
	ErrTokenCantSign                = "can't sign"
	ErrTokenUnexpectedSigningMethod = errors.New(401, "unexpected signing method")
	ErrTokenInvalid                 = errors.New(401, "invalid token")
	ErrUserUnknown                  = errors.New(403, "unknown user")
	ErrMissingScopes                = errors.New(403, "missing scopes")
)

func wrapError(err error) error {
	if errors2.Is(err, jwt.ErrTokenSignatureInvalid) {
		return ErrTokenUnexpectedSigningMethod
	}

	if errors2.Is(err, jwtsvc.ErrTokenInvalid) {
		return ErrTokenInvalid
	}

	if errors2.Is(err, jwtsvc.ErrUserUnknown) {
		return ErrUserUnknown
	}

	if errors2.Is(err, jwtsvc.ErrMissingScopes) {
		return ErrMissingScopes
	}

	return err
}
