package ipsec

import "github.com/vpngen/keydesk/internal/vpn/endpoint"

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"l2tp-username": c.username,
		"l2tp-password": c.password,
	}, nil
}

func (c Config) ConfigureEndpoint(client endpoint.Client) error {
	//TODO implement me
	panic("implement me")
}
