package ipsec

import (
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/internal/vpn"
)

type Generator struct {
	psk, host string
}

func NewGenerator(psk, host string) Generator {
	return Generator{
		psk:  psk,
		host: host,
	}
}

func (g Generator) Generate() (vpn.Config, error) {
	usernameRand := make([]byte, UsernameLen)
	if _, err := rand.Read(usernameRand); err != nil {
		return Config{}, fmt.Errorf("username rand: %w", err)
	}

	passwordRand := make([]byte, PasswordLen)
	if _, err := rand.Read(passwordRand); err != nil {
		return Config{}, fmt.Errorf("password rand: %w", err)
	}

	return Config{
		username: base58.Encode(usernameRand),
		password: base58.Encode(passwordRand),
		host:     g.host,
		psk:      g.psk,
	}, nil
}
