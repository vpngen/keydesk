package ovc

import (
	"fmt"
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
