package keydesk

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/internal/auth"
	"github.com/vpngen/keydesk/keydesk/token"
)

// CreateToken - create JWT.
func CreateToken(authSvc auth.Service, TokenLifeTime int64, scopes []string) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		tc, err := token.New(int(TokenLifeTime))
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		// Create a new jwtoken object, specifying signing method and the claims
		// you would like it to contain.
		jwtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, authSvc.NewClaims(
			scopes,
			tc.Exp(),
			tc.Jti(),
		))

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := jwtoken.SignedString(tc.Secret())
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &tokenString})
	}
}
