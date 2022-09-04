package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vpngen/keykeeper/gen/models"
	"github.com/vpngen/keykeeper/gen/restapi/operations"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang-jwt/jwt"
)

// Tokens errors.
var (
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("token expired")
	ErrUnknownUser             = errors.New("unknown user")

	ErrCantSign = "can't sign"
)

// ValidateBearer - validate our key.
func ValidateBearer(BrigadierID string) func(string) (interface{}, error) {
	return func(bearerHeader string) (interface{}, error) {
		_, bearerToken, ok := strings.Cut(bearerHeader, " ")
		if !ok {
			return nil, fmt.Errorf("decode error")
		}

		claims := jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(bearerToken, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
			}

			jti, _ := claims["jti"].(string)

			return fetchSecret(jti), nil
		})
		if err != nil {
			return nil, fmt.Errorf("parse error: %w", err)
		}

		if !token.Valid {
			return nil, ErrInvalidToken
		}

		if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
			return nil, ErrExpiredToken
		}

		brigadier, _ := claims["user"].(string)

		if brigadier != BrigadierID {
			return nil, ErrUnknownUser
		}

		return brigadier, nil
	}
}

// CreateToken - create JWT.
func CreateToken(BrigadierID string, TokenLifeTime int64) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		_token, err := newToken(int(TokenLifeTime))
		if err != nil {
			return operations.NewPostTokenDefault(500)
		}

		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user": BrigadierID,
			"jti":  _token.jti,
			"exp":  _token.exp.Unix(),
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString(_token.secret)
		if err != nil {
			return operations.NewPostTokenDefault(500)
		}

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &tokenString})
	}
}
