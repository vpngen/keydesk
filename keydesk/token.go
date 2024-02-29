package keydesk

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	jwt2 "github.com/vpngen/keydesk/pkg/jwt"
	"time"
)

// CreateToken - create JWT.
func CreateToken(issuer jwt2.Issuer, ttlSeconds int64) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		claims := issuer.CreateToken(time.Duration(ttlSeconds) * time.Second)
		token, err := issuer.Sign(claims)
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}
		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &token})
	}
}
