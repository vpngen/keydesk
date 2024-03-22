package app

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func SetupServer(db *storage.BrigadeStorage, jwtPubFileName string) (*echo.Echo, error) {
	//pubFile, err := os.Open(jwtPubFileName)
	//if err != nil {
	//	return nil, fmt.Errorf("read jwt public key: %w", err)
	//}
	//
	//ecPub, err := utils.ReadECPublicKey(pubFile)
	//if err != nil {
	//	return nil, fmt.Errorf("decode jwt public key: %w", err)
	//}

	//swagger, err := messages.GetSwagger()
	//if err != nil {
	//	return nil, fmt.Errorf("get swagger: %s", err.Error())
	//}
	//
	//swagger.Servers = nil
	//
	//validator := oapiechomw.OapiRequestValidatorWithOptions(
	//	swagger,
	//	&oapiechomw.Options{
	//		Options: openapi3filter.Options{
	//			AuthenticationFunc: authmw.AuthFuncFactory(jwt.NewAuthorizer(ecPub, jwt.Options{
	//				Issuer:        "dc-mgmt",
	//				Audience:      []string{"keydesk"},
	//				SigningMethod: jwt2.SigningMethodES256,
	//			})),
	//		},
	//	})

	e := echo.New()
	e.HideBanner = true
	logger := echomw.LoggerWithConfig(echomw.LoggerConfig{
		Format:           "${time_custom}\t${method}\t${uri}\t${status}\n",
		CustomTimeFormat: "2006-01-02 15:04:05 -07:00",
	})
	e.Use(echomw.Recover(), logger)

	//e.Use(echomw.Recover(), logger, validator)
	//messages.RegisterHandlers(e, messages.NewStrictHandler(server.NewServer(db, service.New(db)), nil))

	return e, nil
}
