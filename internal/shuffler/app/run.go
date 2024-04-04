package app

import (
	"context"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3filter"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	oapiechomw "github.com/oapi-codegen/echo-middleware"
	"github.com/vpngen/keydesk/gen/shuffler"
	authmw "github.com/vpngen/keydesk/internal/auth/swagger3"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/utils"
	"os"
)

func SetupServer(db *storage.BrigadeStorage, jwtPubFileName string) (*echo.Echo, error) {
	pubFile, err := os.Open(jwtPubFileName)
	if err != nil {
		return nil, fmt.Errorf("read jwt public key: %w", err)
	}

	ecPub, err := utils.ReadECPublicKey(pubFile)
	if err != nil {
		return nil, fmt.Errorf("decode jwt public key: %w", err)
	}

	swagger, err := shuffler.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("get swagger: %s", err.Error())
	}

	swagger.Servers = nil

	validator := oapiechomw.OapiRequestValidatorWithOptions(
		swagger,
		&oapiechomw.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: authmw.AuthFuncFactory(jwt.NewAuthorizer(ecPub, jwt.Options{
					Issuer:        "dc-mgmt",
					Audience:      []string{"keydesk"},
					SigningMethod: jwt2.SigningMethodES256,
				})),
			},
		})

	e := echo.New()
	e.HideBanner = true
	logger := echomw.LoggerWithConfig(echomw.LoggerConfig{
		Format:           "${time_custom}\t${method}\t${uri}\t${status}\n",
		CustomTimeFormat: "2006-01-02 15:04:05 -07:00",
	})

	e.Use(echomw.Recover(), logger, validator)
	shuffler.RegisterHandlers(e, shuffler.NewStrictHandler(server.NewServer(db, service.New(db)), nil))

	return e, nil
}

type server struct {
	s service
}

func (s server) PostConfigs(ctx context.Context, request shuffler.PostConfigsRequestObject) (shuffler.PostConfigsResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s server) GetConfigsSlots(ctx context.Context, request shuffler.GetConfigsSlotsRequestObject) (shuffler.GetConfigsSlotsResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

type service struct {
	db *storage.BrigadeStorage
}

//func (s service) createConfig(cfgType, subdomain string) error {
//
//}
