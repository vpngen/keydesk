package user

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/internal/vpn/ipsec"
	"github.com/vpngen/keydesk/internal/vpn/outline"
	"github.com/vpngen/keydesk/internal/vpn/ovc"
	"github.com/vpngen/keydesk/internal/vpn/wg"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
	"strings"
	"time"
)

type Service struct {
	db                     *storage.BrigadeStorage
	epClient               endpoint.Client
	routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte
}

func New(db *storage.BrigadeStorage) Service {
	var epClient endpoint.Client
	if db.GetActualAddrPort().IsValid() {
		epClient = endpoint.NewClient(db.GetActualAddrPort())
	} else {
		epClient = endpoint.MockClient{
			RealClient: endpoint.NewClient(db.GetCalculatedAddrPort()),
		}
	}
	return Service{
		db:       db,
		epClient: epClient,
	}
}

type User struct {
	Name    string
	Configs map[string]vpn.Config2
}

func (s Service) result(protocols vpn.ProtocolSet) (User, error) {
	f, brigade, err := s.db.OpenDbToModify()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	names, uids, ips4, ips6 := getExisting(*brigade)

	name, person, err := getUniquePerson(names)
	uid := getUniqueUUID(uids)
	ip4 := getUniqueAddr4(brigade.IPv4CGNAT, ips4)
	ip6 := getUniqueAddr6(brigade.IPv6ULA, ips6)
	name = blurIP4(name, brigade.BrigadeID, ip4, brigade.IPv4CGNAT)

	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return User{}, fmt.Errorf("generate wg key: %w", err)
	}

	brigadeUser := newBrigadeUser(uid, name, ip4, ip6, *brigade)

	protocols, unsupported := protocols.GetSupported(brigade.GetSupportedVPNProtocols())
	if unsupported > 0 {
		return User{}, fmt.Errorf("unsupported VPN protocols: %s", unsupported)
	}

	user := User{
		Name:    name,
		Configs: make(map[string]vpn.Config2),
	}

	dbUser := storage.NewUser(uid, name, time.Now(), false, ip4, ip6, person)

	epParams := make(map[string]string)

	for _, protocol := range strings.Split(protocols.String(), ",") {
		var generator vpn.Generator

		switch protocol {
		case vpn.WG:
			generator = getWGGenerator(*brigade, priv, priv.PublicKey(), ip4, ip6, name)
		case vpn.IPSec:
			generator = getIPSecGenerator(*brigade)
		case vpn.Outline:
			generator = newOutlineGenerator(*brigade, name)
		case vpn.OVC:
			g, err := newOVCGenerator(*brigade, name, clientCert, ip4, priv.PublicKey())
			if err != nil {
				return User{}, fmt.Errorf("get OVC generator: %w", err)
			}
			generator = g
		default:
			return User{}, fmt.Errorf("unsupported VPN protocol: %s", protocol)
		}

		config, err := generator.Generate()
		if err != nil {
			return User{}, fmt.Errorf("generate %s: %w", protocol, err)
		}

		protocolClientParams, err := config.GetEndpointParams()
		if err != nil {
			return User{}, fmt.Errorf("get endpoint params for %s: %w", protocol, err)
		}

		for k, v := range protocolClientParams {
			epParams[k] = v
		}

		if err = config.SaveToUser(&dbUser, s.routerPub, s.shufflerPub); err != nil {
			return User{}, fmt.Errorf("save %s to user %s: %w", protocol, name, err)
		}

		//clientConfig, err := config.GetClientConfig()
		//if err != nil {
		//	return User{}, fmt.Errorf("get client config: %w", err)
		//}
		//
		//user.Configs[protocol] = config
		//
		//switch protocol {
		//case vpn.OVC:
		//	caPem, err := kdlib.Unbase64Ungzip(brigade.OvCACertPemGzipBase64)
		//	if err != nil {
		//		return nil, fmt.Errorf("unbase64 ca: %w", err)
		//	}
		//	brigadeUser.OvCACertPem = string(caPem)
		//	brigadeUser.CloakFakeDomain = brigade.CloakFakeDomain
		//case vpn.Outline:
		//	brigadeUser.OutlinePort = brigade.OutlinePort
		//}
	}

	res, err := s.epClient.PeerAdd(priv.PublicKey(), epParams)
	if err != nil {
		return User{}, fmt.Errorf("peer add: %w", err)
	}
}

func getWGGenerator(brigade storage.Brigade, wgPriv, wgPub wgtypes.Key, ip4, ip6 netip.Addr, userName string) vpn.Generator {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}

	return wg.NewGenerator(
		wgPriv,
		wgPub,
		wgtypes.Key(brigade.WgPublicKey),
		ip4,
		ip6,
		brigade.DNSv4,
		brigade.DNSv6,
		host,
		userName,
		brigade.EndpointPort,
	)
}

func newOutlineGenerator(brigade storage.Brigade, userName string) vpn.Generator {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	return outline.NewGenerator(userName, host, brigade.OutlinePort)
}

func getIPSecGenerator(brigade storage.Brigade) vpn.Generator {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	return ipsec.NewGenerator(brigade.IPSecPSK, host)
}

func newOVCGenerator(brigade storage.Brigade, name, clientCert string, ep4 netip.Addr, wgPub wgtypes.Key) (vpn.Generator, error) {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	caPem, err := kdlib.Unbase64Ungzip(brigade.OvCACertPemGzipBase64)
	if err != nil {
		return nil, fmt.Errorf("unbase64 ca: %w", err)
	}
	return ovc.NewGenerator(host, name, brigade.CloakFakeDomain, string(caPem), clientCert, ep4, wgPub), nil
}

func newBrigadeUser(id uuid.UUID, name string, ip4, ip6 netip.Addr, brigade storage.Brigade) storage.UserConfig {
	return storage.UserConfig{
		ID:               id,
		Name:             name,
		IPv4:             ip4,
		IPv6:             ip6,
		EndpointWgPublic: brigade.WgPublicKey,
		EndpointIPv4:     brigade.EndpointIPv4,
		EndpointDomain:   brigade.EndpointDomain,
		EndpointPort:     brigade.EndpointPort,
		DNSv4:            brigade.DNSv4,
		DNSv6:            brigade.DNSv6,
		IPSecPSK:         brigade.IPSecPSK,
	}
}

type set[T comparable] map[T]struct{}

func getExisting(brigade storage.Brigade) (set[string], set[uuid.UUID], set[netip.Addr], set[netip.Addr]) {
	name := make(set[string])
	uid := make(set[uuid.UUID])
	addr4 := map[netip.Addr]struct{}{
		brigade.IPv4CGNAT.Addr():                {},
		kdlib.LastPrefixIPv4(brigade.IPv4CGNAT): {},
	}
	addr6 := map[netip.Addr]struct{}{
		brigade.IPv6ULA.Addr():                {},
		kdlib.LastPrefixIPv6(brigade.IPv6ULA): {},
	}

	for _, user := range brigade.Users {
		name[user.Name] = struct{}{}
		uid[user.UserID] = struct{}{}
		addr4[user.IPv4Addr] = struct{}{}
		addr6[user.IPv6Addr] = struct{}{}
	}

	return name, uid, addr4, addr6
}

func getUniquePerson(nameSet set[string]) (string, namesgenerator.Person, error) {
	for {
		name, person, err := namesgenerator.PeaceAwardeeShort()
		if err != nil {
			return "", person, fmt.Errorf("generate person: %w", err)
		}
		if _, ok := nameSet[name]; !ok {
			return name, person, err
		}
	}
}

func getUniqueUUID(uid set[uuid.UUID]) uuid.UUID {
	for {
		id := uuid.New()
		if _, ok := uid[id]; !ok {
			return id
		}
	}
}

func getUniqueAddr4(ip4CGNAT netip.Prefix, addr4 set[netip.Addr]) netip.Addr {
	for {
		addr := kdlib.RandomAddrIPv4(ip4CGNAT)
		if _, ok := addr4[addr]; !ok {
			return addr
		}
	}
}

func getUniqueAddr6(ip6ULA netip.Prefix, addr6 set[netip.Addr]) netip.Addr {
	for {
		addr := kdlib.RandomAddrIPv6(ip6ULA)
		if _, ok := addr6[addr]; !ok {
			return addr
		}
	}
}

func blurIP4(name, brigadeID string, ip4 netip.Addr, ip4CGNAT netip.Prefix) string {
	return fmt.Sprintf(
		"%03d %s",
		kdlib.BlurIpv4Addr(ip4, ip4CGNAT.Bits(), kdlib.ExtractUint32Salt(brigadeID)),
		name,
	)
}
