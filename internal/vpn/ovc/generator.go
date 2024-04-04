package ovc

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/kdlib"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

type generator struct {
	host, name, dns1, dns2, fakeDomain, caCert, clientCert string
	ep4                                                    netip.Addr
	wgPub                                                  wgtypes.Key
}

func NewGenerator(host, name, fakeDomain, caCert, clientCert string, ep4 netip.Addr, wgPub wgtypes.Key) generator {
	return generator{host: host, name: name, dns1: defaultInternalDNS, dns2: defaultInternalDNS, fakeDomain: fakeDomain, caCert: caCert, clientCert: clientCert, ep4: ep4, wgPub: wgPub}
}

func (g generator) Generate() (vpn.Config2, error) {
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
		clientCert: g.clientCert,
		ep4:        g.ep4,
		wgPub:      g.wgPub,
	}, nil

}
