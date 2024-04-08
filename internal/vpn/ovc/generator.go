package ovc

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/kdlib"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

type Generator struct {
	host, name, dns1, dns2, fakeDomain, caCert string
	ep4                                        netip.Addr
	wgPub                                      wgtypes.Key
}

func NewGenerator(host, name, fakeDomain, caCert string, ep4 netip.Addr, wgPub wgtypes.Key) Generator {
	return Generator{host: host, name: name, dns1: defaultInternalDNS, dns2: defaultInternalDNS, fakeDomain: fakeDomain, caCert: caCert, ep4: ep4, wgPub: wgPub}
}

func (g Generator) Generate() (vpn.Config, error) {
	cn := uuid.New()
	csr, key, err := kdlib.NewOvClientCertRequest(cn.String())
	if err != nil {
		return nil, fmt.Errorf("ov new csr: %w", err)
	}
	return Config{
		cn:         cn,
		bypass:     uuid.New(),
		key:        key,
		csr:        csr,
		host:       g.host,
		name:       g.name,
		dns1:       g.dns1,
		dns2:       g.dns2,
		fakeDomain: g.fakeDomain,
		caCert:     g.caCert,
		ep4:        g.ep4,
		wgPub:      g.wgPub,
	}, nil

}
