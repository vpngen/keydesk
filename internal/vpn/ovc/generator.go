package ovc

import (
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

type Generator struct {
	host, name, dns1, dns2, fakeDomain, caCert string
	ep4                                        netip.Addr
	wgPub                                      wgtypes.Key
}

func NewGenerator(
	host, name, fakeDomain, caCert string,
	ep4 netip.Addr,
	wgPub wgtypes.Key,
) Generator {
	return Generator{
		host:       host,
		name:       name,
		dns1:       defaultInternalDNS,
		dns2:       defaultInternalDNS,
		fakeDomain: fakeDomain,
		caCert:     caCert,
		ep4:        ep4,
		wgPub:      wgPub,
	}
}

func (g Generator) Generate(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (vpn.Config, error) {
	cn := uuid.New()
	bypass := uuid.New()
	csr, key, err := kdlib.NewOvClientCertRequest(cn.String())
	if err != nil {
		return nil, fmt.Errorf("ov new csr: %w", err)
	}

	routerBypass, err := box.SealAnonymous(nil, bypass[:], &routerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cloakBypassUID router seal: %w", err)
	}

	shufflerBypass, err := box.SealAnonymous(nil, bypass[:], &shufflerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cloakBypassUID shuffler seal: %w", err)
	}

	return Config{
		cn:             cn,
		bypass:         bypass,
		key:            key,
		csr:            csr,
		routerBypass:   routerBypass,
		shufflerBypass: shufflerBypass,
		host:           g.host,
		name:           g.name,
		dns1:           g.dns1,
		dns2:           g.dns2,
		fakeDomain:     g.fakeDomain,
		caCert:         g.caCert,
		ep4:            g.ep4,
		wgPub:          g.wgPub,
	}, nil
}
