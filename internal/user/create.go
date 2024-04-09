package user

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/keydesk/storage"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"strings"
	"time"
)

type User struct {
	UUID    uuid.UUID
	Configs map[string]any
}

type createUserResponse struct {
	User
	FreeSlots, TotalSlots uint
}

func (s Service) CreateUser(protocols vpn.ProtocolSet) (res createUserResponse, err error) {
	err = s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		res.User, err = s.createUserWithConfigs(protocols, brigade)
		if err != nil {
			return fmt.Errorf("create user with configs %s: %w", protocols, err)
		}

		res.FreeSlots, res.TotalSlots = s.getSlotsInfo(brigade)

		return nil
	})

	return
}

func (s Service) createUserWithConfigs(protocols vpn.ProtocolSet, brigade *storage.Brigade) (User, error) {
	protocols |= vpn.TypeWG // wg is always required

	protocols, unsupported := protocols.GetSupported(vpn.NewProtocolSet(brigade.GetSupportedVPNProtocols()))
	if unsupported > 0 {
		return User{}, fmt.Errorf("unsupported VPN protocols: %s", unsupported)
	}

	dbUser, err := newUser(brigade)
	if err != nil {
		return User{}, fmt.Errorf("new user: %w", err)
	}

	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return User{}, fmt.Errorf("generate wg key: %w", err)
	}

	configs, err := s.generateConfigs(protocols, *brigade, dbUser, priv, priv.PublicKey())
	if err != nil {
		return User{}, fmt.Errorf("generate configs %s: %w", protocols, err)
	}

	epParams, err := getEndpointParams(configs)
	if err != nil {
		return User{}, fmt.Errorf("get endpoint params: %w", err)
	}

	// call endpoint api
	epDdata, err := s.epClient.PeerAdd(priv.PublicKey(), epParams)
	if err != nil {
		return User{}, fmt.Errorf("peer add: %w", err)
	}

	if err = saveConfigsToDB(brigade, &dbUser, configs); err != nil {
		return User{}, fmt.Errorf("save configs to db: %w", err)
	}

	clientCfgs, err := getClientConfigs(configs, epDdata)
	if err != nil {
		return User{}, fmt.Errorf("get client configs: %w", err)
	}

	return User{
		UUID:    dbUser.UserID,
		Configs: clientCfgs,
	}, nil
}

func newUser(brigade *storage.Brigade) (storage.User, error) {
	names, uids, ips4, ips6 := getExisting(brigade)

	name, person, err := getUniquePerson(names)
	if err != nil {
		return storage.User{}, fmt.Errorf("get unique person: %w", err)
	}

	uid := getUniqueUUID(uids)
	ip4 := getUniqueAddr4(brigade.IPv4CGNAT, ips4)
	ip6 := getUniqueAddr6(brigade.IPv6ULA, ips6)
	name = blurIP4(name, brigade.BrigadeID, ip4, brigade.IPv4CGNAT)

	return storage.NewUser(uid, name, time.Now(), false, ip4, ip6, person), nil
}

func (s Service) generateConfigs(protocols vpn.ProtocolSet, brigade storage.Brigade, user storage.User, wgPriv, wgPub wgtypes.Key) (map[string]vpn.Config, error) {
	configs := make(map[string]vpn.Config)
	for _, protocol := range strings.Split(protocols.String(), ",") {
		generator, err := newGenerator(protocol, brigade, user, wgPriv, wgPub)
		if err != nil {
			return nil, fmt.Errorf("get %s generator: %w", protocol, err)
		}

		config, err := generator.Generate(s.routerPub, s.shufflerPub)
		if err != nil {
			return nil, fmt.Errorf("generate %s: %w", protocol, err)
		}

		configs[protocol] = config
	}

	return configs, nil
}

func getEndpointParams(configs map[string]vpn.Config) (map[string]string, error) {
	epParams := make(map[string]string)
	for protocol, config := range configs {
		protocolClientParams, err := config.GetEndpointParams()
		if err != nil {
			return nil, fmt.Errorf("get endpoint params for %s: %w", protocol, err)
		}
		for k, v := range protocolClientParams {
			epParams[k] = v
		}
	}
	return epParams, nil
}

func saveConfigsToDB(brigade *storage.Brigade, user *storage.User, configs map[string]vpn.Config) error {
	for protocol, config := range configs {
		if err := config.Store(user); err != nil {
			return fmt.Errorf("save %s config to user %s: %w", protocol, user.Name, err)
		}
	}

	brigade.Users = append(brigade.Users, user)

	return nil
}

func getClientConfigs(configs map[string]vpn.Config, epData endpoint.APIResponse) (map[string]any, error) {
	clientCfgs := make(map[string]any)
	for protocol, config := range configs {
		clientCfg, err := config.GetClientConfig(epData)
		if err != nil {
			return nil, fmt.Errorf("get %s client config: %w", protocol, err)
		}
		clientCfgs[protocol] = clientCfg
	}
	return clientCfgs, nil
}
