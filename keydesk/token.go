package keydesk

import (
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/internal/auth"
	"github.com/vpngen/keydesk/keydesk/token"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// CreateToken - create JWT.
func CreateToken(BrigadierID string, TokenLifeTime int64, scopes []string) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		tc, err := token.New(int(TokenLifeTime))
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		now := time.Now()

		// Create a new jwtoken object, specifying signing method and the claims
		// you would like it to contain.
		jwtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "keydesk",
				Subject:   BrigadierID,
				Audience:  []string{"keydesk"},
				ExpiresAt: jwt.NewNumericDate(tc.Exp()),
				NotBefore: jwt.NewNumericDate(now),
				IssuedAt:  jwt.NewNumericDate(now),
				ID:        tc.Jti(),
			},
			Scopes: scopes,
			User:   BrigadierID,
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := jwtoken.SignedString(tc.Secret())
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &tokenString})
	}
}
