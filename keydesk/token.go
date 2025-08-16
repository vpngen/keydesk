package keydesk

import (
	"fmt"
	"os"
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
		claims := issuer.CreateToken(time.Duration(ttlSeconds)*time.Second, db.IsVIP())

		token, err := issuer.Sign(claims)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sign token: %s\n", err)

			return operations.NewPostTokenInternalServerError()
		}

		fmt.Fprintf(os.Stderr, "token created: %s\n", token)

		return operations.NewPostTokenCreated().WithPayload(&models.Token{Token: &token})
	}
}
