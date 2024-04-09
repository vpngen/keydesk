package wg

import (
	"crypto/rand"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

type Generator struct {
	priv, pub, epPub     wgtypes.Key
	ip4, ip6, dns4, dns6 netip.Addr
	host, userName       string
	port                 uint16
}

func NewGenerator(
	priv, pub, epPub wgtypes.Key,
	ip4, ip6, dns4, dns6 netip.Addr,
	host, userName string,
	port uint16,
) Generator {
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

func (g Generator) Generate(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (vpn.Config, error) {
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("psk: %w", err)
	}

	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("priv: %w", err)
	}

	routerPsk, err := box.SealAnonymous(nil, psk[:], &routerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("psk router seal: %w", err)
	}

	shufflerPsk, err := box.SealAnonymous(nil, psk[:], &shufflerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("psk shuffler seal: %w", err)
	}

	return Config{
		pub:         priv.PublicKey(),
		priv:        priv,
		psk:         psk,
		routerPSK:   routerPsk,
		shufflerPSK: shufflerPsk,
		epPub:       g.epPub,
		ip4:         g.ip4,
		ip6:         g.ip6,
		dns4:        g.dns4,
		dns6:        g.dns6,
		host:        g.host,
		userName:    g.userName,
		port:        g.port,
	}, nil
}
