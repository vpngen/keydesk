package jwt

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/vpngen/keydesk/pkg/jwt"
)

func AuthFuncFactory(authorizer jwt.MessagesJwtAuthorizer) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		switch input.SecuritySchemeName {
		case "JWTAuth":
			tokenStr := strings.TrimPrefix(
				input.RequestValidationInput.Request.Header.Get("Authorization"),
				"Bearer ",
			)
			claims, err := authorizer.Validate(tokenStr)
			if err != nil {
				return err
			}
			return authorizer.Authorize(claims, input.Scopes...)
		default:
			return fmt.Errorf("security scheme %s is not supported", input.SecuritySchemeName)
		}
	}
}
