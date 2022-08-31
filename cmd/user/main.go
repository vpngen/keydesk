package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"

	"test/gen/models"
	"test/gen/restapi"
	"test/gen/restapi/operations"

	"github.com/golang-jwt/jwt"
)

func main() {
	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	// create new service API
	api := operations.NewUserAPI(swaggerSpec)
	server := restapi.NewServer(api)
	defer server.Shutdown()

	server.Port = 8080

	// TODO: Set Handle

	api.BearerAuth = ValidateBearer
	api.PostTokenHandler = operations.PostTokenHandlerFunc(CreateToken)

	server.ConfigureAPI()

	// serve API
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}
}

// Tokens errors.
var (
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrInvalidToken            = errors.New("invalid token")
)

// MySecretKeyForJWT - moke.
const MySecretKeyForJWT = "барракуда"

// ValidateBearer - validate our key.
func ValidateBearer(bearerHeader string) (interface{}, error) {
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

	if token.Valid {
		return claims["user"].(string), nil
	}
	return nil, ErrInvalidToken
}

// CreateToken - creaste JWT.
func CreateToken(params operations.PostTokenParams) middleware.Responder {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": "bar",
		"nbf":  time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(MySecretKeyForJWT))
	if err != nil {
		return operations.NewPostTokenDefault(500)
	}

	return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &tokenString})
}
