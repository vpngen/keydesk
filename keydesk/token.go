package keydesk

import (
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk/storage"
	jwtsvc "github.com/vpngen/keydesk/pkg/jwt"
)

// CreateToken - create JWT.
func CreateToken(db *storage.BrigadeStorage, issuer jwtsvc.KeydeskTokenIssuer, ttlSeconds int64) func(operations.PostTokenParams) middleware.Responder {
	return func(params operations.PostTokenParams) middleware.Responder {
		claims := issuer.CreateKeydeskToken(time.Duration(ttlSeconds)*time.Second, db.IsVIP())

		token, err := issuer.SignKeydeskToken(claims)
		if err != nil {
			return operations.NewPostTokenInternalServerError()
		}

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &token})
	}
}
