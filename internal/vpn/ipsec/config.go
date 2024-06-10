package ipsec

import (
	"github.com/vpngen/keydesk/internal/vpn"
)

const (
	UsernameLen = 16
	PasswordLen = 32
)

// Config implements vpn.Config
type Config struct {
	username, password, host, psk                      string
	routerUser, routerPass, shufflerUser, shufflerPass []byte
}

func (c Config) Protocol() string {
	return vpn.IPSec
}
