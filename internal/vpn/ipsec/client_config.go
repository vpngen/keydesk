package ipsec

import "github.com/vpngen/keydesk/internal/vpn/endpoint"

type ClientConfig struct {
	Username string
	Password string
	Host     string
	PSK      string
}

func (c Config) GetClientConfig(_ endpoint.APIResponse) (any, error) {
	return ClientConfig{
		Username: c.username,
		Password: c.password,
		Host:     c.host,
		PSK:      c.psk,
	}, nil
}
