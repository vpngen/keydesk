package ipsec

import (
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

// Generator implements vpn.Generator
type Generator struct {
	psk, host string
}

func NewGenerator(psk, host string) Generator {
	return Generator{
		psk:  psk,
		host: host,
	}
}

func (g Generator) Generate(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (vpn.Config, error) {
	usernameRand := make([]byte, UsernameLen)
	if _, err := rand.Read(usernameRand); err != nil {
		return Config{}, fmt.Errorf("username rand: %w", err)
	}

	passwordRand := make([]byte, PasswordLen)
	if _, err := rand.Read(passwordRand); err != nil {
		return Config{}, fmt.Errorf("password rand: %w", err)
	}

	username := base58.Encode(usernameRand)[:UsernameLen]
	password := base58.Encode(passwordRand)[:UsernameLen]

	routerUser, err := box.SealAnonymous(nil, []byte(username), &routerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("username router seal: %w", err)
	}

	routerPass, err := box.SealAnonymous(nil, []byte(password), &routerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("password router seal: %w", err)
	}

	shufflerUser, err := box.SealAnonymous(nil, []byte(username), &shufflerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("username shuffler seal: %w", err)
	}

	shufflerPass, err := box.SealAnonymous(nil, []byte(password), &shufflerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("password shuffler seal: %w", err)
	}

	return Config{
		username:     username,
		password:     password,
		host:         g.host,
		psk:          g.psk,
		routerUser:   routerUser,
		routerPass:   routerPass,
		shufflerUser: shufflerUser,
		shufflerPass: shufflerPass,
	}, nil
}
