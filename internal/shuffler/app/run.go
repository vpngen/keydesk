package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	oapiechomw "github.com/oapi-codegen/echo-middleware"
	"github.com/vpngen/keydesk/gen/shuffler"
	authmw "github.com/vpngen/keydesk/internal/auth/swagger3"
	"github.com/vpngen/keydesk/internal/user"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/ipsec"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/vpngine/naclkey"
	"log"
	"net/http"
	"os"
	"strings"
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
	protocols := vpn.NewProtocolSetFromString(string(request.Body.Type))
	if len(protocols.Protocols()) != 1 {
		return shuffler.PostConfigsdefaultJSONResponse{
			Body:       fmt.Sprintf("type must exactly one of: %s", strings.Join(protocols.Protocols(), ", ")),
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	domain := ""
	if request.Body.Domain != nil {
		domain = *request.Body.Domain
	}

	userCfg, err := s.service.CreateUser(protocols, domain)
	if err != nil {
		return shuffler.PostConfigsdefaultJSONResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	protocol := protocols.String()

	res := shuffler.PostConfigs201JSONResponse{
		FreeSlots: int(userCfg.FreeSlots),
		Id:        userCfg.UUID,
		Type:      shuffler.ConfigType(protocol),
	}

	res.Config, err = getVPNConfig(protocol, userCfg.Configs[protocol])
	if err != nil {
		return shuffler.PostConfigsdefaultJSONResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
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

func getVPNConfig(protocol string, data any) (shuffler.VPNConfig, error) {
	switch protocol {
	default:
		return shuffler.VPNConfig{}, fmt.Errorf("unsupported protocol: %s", protocol)
	case vpn.WG:
		return getWGConfig(data)
	case vpn.OVC:
		return getOVCConfig(data)
	case vpn.IPSec:
		return getIPSecConfig(data)
	case vpn.Outline:
		return getOutlineConfig(data)
	}
}

func getWGConfig(data any) (shuffler.VPNConfig, error) {
	cfg, ok := data.(vpn.FileConfig)
	if !ok {
		return shuffler.VPNConfig{}, fmt.Errorf("wg: expected vpn.FileConfig, got %T", data)
	}

	ret := shuffler.VPNConfig{}

	if err := ret.FromWireGuardConfig(shuffler.WireGuardConfig{
		FileContent: cfg.Content,
		FileName:    cfg.FileName,
		TunnelName:  cfg.ConfigName,
	}); err != nil {
		return shuffler.VPNConfig{}, fmt.Errorf("wg: %w", err)
	}

	return ret, nil
}

func getOVCConfig(data any) (shuffler.VPNConfig, error) {
	cfg, ok := data.(vpn.FileConfig)
	if !ok {
		return shuffler.VPNConfig{}, fmt.Errorf("ovc: expected vpn.FileConfig, got %T", data)
	}

	ret := shuffler.VPNConfig{}

	if err := ret.FromAmneziaOVCConfig(shuffler.AmneziaOVCConfig{
		FileContent: cfg.Content,
		FileName:    cfg.FileName,
		TunnelName:  cfg.ConfigName,
	}); err != nil {
		return shuffler.VPNConfig{}, fmt.Errorf("ovc: %w", err)
	}

	return ret, nil
}

func getIPSecConfig(data any) (shuffler.VPNConfig, error) {
	cfg, ok := data.(ipsec.ClientConfig)
	if !ok {
		return shuffler.VPNConfig{}, fmt.Errorf("ipsec: expected ipsec.ClientConfig, got %T", data)
	}

	ret := shuffler.VPNConfig{}

	if err := ret.FromIPSecL2TPConfig(shuffler.IPSecL2TPConfig{
		Password: cfg.Password,
		Psk:      cfg.PSK,
		Server:   cfg.Host,
		Username: cfg.Username,
	}); err != nil {
		return shuffler.VPNConfig{}, fmt.Errorf("ipsec: %w", err)
	}

	return ret, nil
}

func getOutlineConfig(data any) (shuffler.VPNConfig, error) {
	cfg, ok := data.(string)
	if !ok {
		return shuffler.VPNConfig{}, fmt.Errorf("outline: expected string, got %T", data)
	}

	ret := shuffler.VPNConfig{}

	if err := ret.FromOutlineConfig(shuffler.OutlineConfig{
		AccessKey: cfg,
	}); err != nil {
		return shuffler.VPNConfig{}, fmt.Errorf("outline: %w", err)
	}

	return ret, nil
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
