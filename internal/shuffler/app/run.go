package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	oapiechomw "github.com/oapi-codegen/echo-middleware"
	"github.com/vpngen/keydesk/gen/shuffler"
	authmw "github.com/vpngen/keydesk/internal/auth/swagger3"
	"github.com/vpngen/keydesk/internal/user"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/vpngine/naclkey"
)

func SetupServer(db *storage.BrigadeStorage, authorizer jwt.Authorizer, routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (*echo.Echo, error) {
	swagger, err := shuffler.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("get swagger: %s", err.Error())
	}

	swagger.Servers = nil

	validator := oapiechomw.OapiRequestValidatorWithOptions(
		swagger,
		&oapiechomw.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: authmw.AuthFuncFactory(authorizer),
			},
		})

	e := echo.New()
	e.HideBanner = true

	loggerMW := echomw.LoggerWithConfig(echomw.LoggerConfig{
		Format:           "${time_custom}\t${method}\t${uri}\t${status}\n",
		CustomTimeFormat: "2006-01-02 15:04:05 -07:00",
	})

	e.Use(echomw.Recover(), loggerMW, validator)

	logger := log.New(os.Stderr, "[endpoint client]\t", log.LstdFlags|log.Lshortfile|log.Lmsgprefix)
	userSvc, err := user.New(db, routerPub, shufflerPub, logger)
	if err != nil {
		return nil, fmt.Errorf("init user service: %w", err)
	}

	srv := server{service: userSvc}

	shuffler.RegisterHandlers(e, shuffler.NewStrictHandler(srv, nil))

	return e, nil
}

type server struct {
	service user.Service
}

func (s server) GetActivity(ctx context.Context, request shuffler.GetActivityRequestObject) (shuffler.GetActivityResponseObject, error) {
	lastSeen, err := s.service.GetLastConnections()
	if err != nil {
		return shuffler.GetActivitydefaultJSONResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}
	return shuffler.GetActivity200JSONResponse(lastSeen), nil
}

func (s server) PostConfigs(ctx context.Context, request shuffler.PostConfigsRequestObject) (shuffler.PostConfigsResponseObject, error) {
	//protocols := vpn.NewProtocolSet(request.Body.Protocols)
	//if len(protocols.Protocols()) != 1 {
	//	return shuffler.PostConfigsdefaultJSONResponse{
	//		Body:       fmt.Sprintf("type must exactly one of: %s", strings.Join(protocols.Protocols(), ", ")),
	//		StatusCode: http.StatusBadRequest,
	//	}, nil
	//}

	domain := ""
	if request.Body.Domain != nil {
		domain = *request.Body.Domain
	}

	userCfg, err := s.service.CreateUser(request.Body.Configs, domain)
	if err != nil {
		return shuffler.PostConfigsdefaultJSONResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	res := shuffler.PostConfigs201JSONResponse{
		FreeSlots: int(userCfg.FreeSlots),
		Id:        userCfg.UUID,
		Name:      userCfg.Name,
		Domain:    userCfg.Domain,
	}

	cfgs := userCfg.Configs

	if cfgs.WireGuard != nil {
		res.Configs.Wireguard = &shuffler.WireGuardConfig{
			FileContent: cfgs.WireGuard.Content,
			FileName:    cfgs.WireGuard.FileName,
			TunnelName:  cfgs.WireGuard.ConfigName,
		}
	}

	if cfgs.Amnezia != nil {
		res.Configs.Amnezia = &shuffler.AmneziaOVCConfig{
			FileContent: cfgs.Amnezia.Content,
			FileName:    cfgs.Amnezia.FileName,
			TunnelName:  cfgs.Amnezia.ConfigName,
		}
	}

	if cfgs.Universal != nil {
		res.Configs.Vgc = cfgs.Universal
	}

	if cfgs.Outline != nil {
		res.Configs.Outline = cfgs.Outline
	}

	if cfgs.Proto0 != nil {
		res.Configs.Proto0 = &shuffler.Proto0Config{
			AccessKey: *cfgs.Proto0,
		}
	}

	if cfgs.IPSec != nil {
		res.Configs.Ipsec = &shuffler.IPSecL2TPConfig{
			Password: cfgs.IPSec.Password,
			Psk:      cfgs.IPSec.PSK,
			Server:   cfgs.IPSec.Host,
			Username: cfgs.IPSec.Username,
		}
	}

	return res, nil
}

func (s server) DeleteConfigsId(ctx context.Context, request shuffler.DeleteConfigsIdRequestObject) (shuffler.DeleteConfigsIdResponseObject, error) {
	free, err := s.service.DeleteUser(request.Id)
	if errors.Is(err, user.ErrNotFound) {
		return shuffler.DeleteConfigsId404Response{}, nil
	} else if err != nil {
		return shuffler.DeleteConfigsIddefaultJSONResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return shuffler.DeleteConfigsId200JSONResponse{FreeSlots: int(free)}, nil
}

func (s server) GetSlots(ctx context.Context, request shuffler.GetSlotsRequestObject) (shuffler.GetSlotsResponseObject, error) {
	free, total, err := s.service.GetSlotsInfo()
	if err != nil {
		return shuffler.GetSlotsdefaultJSONResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return shuffler.GetSlots200JSONResponse{
		FreeSlots:  int(free),
		TotalSlots: int(total),
	}, nil
}
