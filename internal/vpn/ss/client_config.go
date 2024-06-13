package ss

import "github.com/vpngen/keydesk/internal/vpn/endpoint"

func (c Config) GetClientConfig(_ endpoint.APIResponse) (any, error) {
	return ClientConfig{
		Host:     c.host,
		Port:     int(c.port),
		Cipher:   Cipher,
		Password: c.secret,
	}, nil
}

type ClientConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Cipher   string `json:"cipher"`
	Password string `json:"password"`
}

const Cipher = "AEAD_CHACHA20_POLY1305"
