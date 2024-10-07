package wg

import (
	"fmt"
	"net/netip"

	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func Generate(brigade *storage.Brigade, user *storage.User, nacl utils.NaCl, epData map[string]string) (RawConfig, error) {
	// generate keys
	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return RawConfig{}, fmt.Errorf("endpoint pub: %w", err)
	}

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return RawConfig{}, fmt.Errorf("generate key: %w", err)
	}

	psk, err := wgtypes.GenerateKey()
	if err != nil {
		return RawConfig{}, fmt.Errorf("generate psk: %w", err)
	}

	// assemble config
	wgcfg := RawConfig{
		Key: key,
		Address: []netip.Prefix{
			netip.PrefixFrom(user.IPv4Addr, 32),
			netip.PrefixFrom(user.IPv6Addr, 128),
		},
		DNS:          []netip.Addr{brigade.DNSv4, brigade.DNSv6},
		PublicKey:    epPub,
		PresharedKey: psk,
		AllowedIPs: []netip.Prefix{
			netip.PrefixFrom(netip.AddrFrom4([4]byte{}), 0),
			netip.PrefixFrom(netip.AddrFrom16([16]byte{}), 0),
		},
		EndpointHost: storage.GetEndpointHost(brigade, user),
		EndpointPort: brigade.EndpointPort,
	}

	// encrypt
	pskenc, err := nacl.Seal(psk[:])
	if err != nil {
		return RawConfig{}, fmt.Errorf("encrypt psk: %w", err)
	}

	// add endpoint data
	epData["wg-public-key"] = epPub.String()
	epData["wg-psk-key"] = pskenc.Router.Base64()
	epData["allowed-ips"] = wgcfg.GetAddresses()

	// add user data
	pkey := key.PublicKey()
	user.WgPublicKey = pkey[:]
	user.WgPSKRouterEnc = pskenc.Router
	user.WgPSKShufflerEnc = pskenc.Shuffler

	return wgcfg, nil
}
