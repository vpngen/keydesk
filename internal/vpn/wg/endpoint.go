package wg

import (
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"strings"
)

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"wg-public-key": base64.StdEncoding.EncodeToString(c.pub[:]),
		"wg-psk-key":    base64.StdEncoding.EncodeToString(c.priv[:]),
		"allowed-ips":   strings.Join([]string{c.ip4.String(), c.ip6.String()}, ","),
	}, nil
}

func (c Config) ConfigureEndpoint(client endpoint.Client) error {
	params, err := c.GetEndpointParams()
	if err != nil {
		return fmt.Errorf("get endpoint params: %w", err)
	}
	_, err = client.PeerAdd(c.pub, params)
	if err != nil {
		return fmt.Errorf("peer add: %w", err)
	}
	return nil
}
