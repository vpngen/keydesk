package wg

import (
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

type Generator struct {
	priv, pub, epPub     wgtypes.Key
	ip4, ip6, dns4, dns6 netip.Addr
	host, userName       string
	port                 uint16
}

func NewGenerator(priv, pub, epPub wgtypes.Key, ip4, ip6, dns4, dns6 netip.Addr, host, userName string, port uint16) Generator {
	return Generator{
		priv:     priv,
		pub:      pub,
		epPub:    epPub,
		ip4:      ip4,
		ip6:      ip6,
		dns4:     dns4,
		dns6:     dns6,
		host:     host,
		userName: userName,
		port:     port,
	}
}

func (g Generator) Generate() (vpn.Config, error) {
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("psk: %w", err)
	}

	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("priv: %w", err)
	}

	return Config{
		pub:      priv.PublicKey(),
		priv:     priv,
		psk:      psk,
		epPub:    wgtypes.Key{},
		ip4:      netip.Addr{},
		ip6:      netip.Addr{},
		dns4:     netip.Addr{},
		dns6:     netip.Addr{},
		host:     g.host,
		userName: g.userName,
		port:     g.port,
	}, nil
}
