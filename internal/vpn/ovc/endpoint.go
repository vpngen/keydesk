package ovc

import (
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
)

func (c Config) GetEndpointParams() (map[string]string, error) {
	csr, err := c.csrPemGzBase64()
	if err != nil {
		return nil, fmt.Errorf("csr: %w", err)
	}
	return map[string]string{
		"openvpn-client-csr": string(csr),
		"cloak-uid":          c.bypass.String(),
	}, nil
}

func (c Config) ConfigureEndpoint(client endpoint.Client) error {
	//TODO implement me
	panic("implement me")
}
