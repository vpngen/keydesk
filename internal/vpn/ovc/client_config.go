package ovc

import (
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/amnezia"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/kdlib"
)

func (c Config) GetClientConfig(data endpoint.APIResponse) (any, error) {
	// TODO: check if hosts are equal
	amnz := amnezia.NewConfig(c.host, c.name, c.dns1, c.dns2)
	container, err := c.getAmneziaContainer(c.host, c.ep4, c.epPub, c.fakeDomain, c.caCert, data.OpenvpnClientCertificate)
	if err != nil {
		return nil, fmt.Errorf("amnezia container: %w", err)
	}
	amnz.AddContainer(container)
	amnz.SetDefaultContainer(amnezia.ContainerOpenVPNCloak)

	amnzConf, err := amnz.Marshal()
	if err != nil {
		return nil, fmt.Errorf("amnezia marshal: %w", err)
	}

	name := kdlib.AssembleWgStyleTunName(c.name)

	return vpn.FileConfig{
		Content:    amnzConf,
		FileName:   name + ".vpn",
		ConfigName: name,
	}, nil
}
