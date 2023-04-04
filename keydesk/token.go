package keydesk

import (
	"fmt"
	"strings"
	"time"

	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk/token"

	"github.com/go-openapi/errors"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang-jwt/jwt"
)

// Tokens errors.
var (
	ErrTokenUnexpectedSigningMethod = errors.New(401, "unexpected signing method")
	ErrTokenInvalid                 = errors.New(401, "invalid token")
	ErrTokenExpired                 = errors.New(403, "token expired")
	ErrUserUnknown                  = errors.New(403, "unknown user")

	ErrTokenCantSign = "can't sign"
)

// ValidateBearer - validate our key.
func ValidateBearer(BrigadierID string) func(string) (interface{}, error) {
	return func(bearerHeader string) (interface{}, error) {
		_, bearerToken, ok := strings.Cut(bearerHeader, " ")
		if !ok {
			return nil, ErrTokenInvalid
		}

		claims := jwt.MapClaims{}

		jwtoken, err := jwt.ParseWithClaims(bearerToken, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrTokenUnexpectedSigningMethod, t.Header["alg"])
			}

			jti, _ := claims["jti"].(string)

			return token.FetchSecret(jti), nil
		})
		if err != nil {
			return nil, ErrTokenInvalid
		}

		if !jwtoken.Valid {
			return nil, ErrTokenInvalid
		}

		if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
			return nil, ErrTokenExpired
		}

		brigadier, _ := claims["user"].(string)

		if brigadier != BrigadierID {
			return nil, ErrUserUnknown
		}

		return brigadier, nil
	}
}

// CreateToken - create JWT.
func CreateToken(BrigadierID string, TokenLifeTime int64) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		tc, err := token.New(int(TokenLifeTime))
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		// Create a new jwtoken object, specifying signing method and the claims
		// you would like it to contain.
		jwtoken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user": BrigadierID,
			"jti":  tc.Jti(),
			"exp":  tc.Exp().Unix(),
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := jwtoken.SignedString(tc.Secret())
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &tokenString})
	}
}
