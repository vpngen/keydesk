package token

import (
	"errors"
	"fmt"
	"strings"
	"test/gen/models"
	"test/gen/restapi/operations"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

// Tokens errors.
var (
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("token expired")
	ErrUnknownUser             = errors.New("unknown user")

	ErrCantSign = "can't sign"
)

// MySecretKeyForJWT - moke.
const MySecretKeyForJWT = "барракуда"

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
			return []byte(MySecretKeyForJWT), nil
		})
		if err != nil {
			return nil, fmt.Errorf("parse error: %w", err)
		}

		if !token.Valid {
			return nil, ErrInvalidToken
		}

		if claims.VerifyExpiresAt(time.Now().Unix(), false) {
			return nil, ErrExpiredToken
		}

		brigadier := claims["user"].(string)

		if brigadier != BrigadierID {
			return nil, ErrUnknownUser
		}

		return brigadier, nil
	}
}

// CreateToken - creaste JWT.
func CreateToken(BrigadierID string, TokenLifeTime int64) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user": BrigadierID,
			"jti":  uuid.New().String(),
			"exp":  time.Now().Unix() + TokenLifeTime,
		})

		// Sign and get the complete encoded token as a string using the secret
		tokenString, err := token.SignedString([]byte(MySecretKeyForJWT))
		if err != nil {
			return operations.NewPostTokenDefault(500).WithPayload(&models.Error{Message: &ErrCantSign})
		}

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &tokenString})
	}
}
