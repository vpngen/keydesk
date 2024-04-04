package ipsec

import (
	"github.com/vpngen/keydesk/internal/vpn"
)

const (
	UsernameLen = 16
	PasswordLen = 32
)

type Config struct {
	username, password, host, psk string
}

func (c Config) Protocol() string {
	return vpn.IPSec
}
