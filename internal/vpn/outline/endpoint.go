package outline

import "github.com/vpngen/keydesk/internal/vpn/endpoint"

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"outline-ss-password": c.secret,
	}, nil
}

func (c Config) ConfigureEndpoint(client endpoint.Client) error {
	//TODO implement me
	panic("implement me")
}
