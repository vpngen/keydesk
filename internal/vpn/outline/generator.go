package outline

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
	name, host string
	port       uint16
}

func NewGenerator(name, host string, port uint16) Generator {
	return Generator{
		name: name,
		host: host,
		port: port,
	}
}

func (g Generator) Generate(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (vpn.Config, error) {
	secretRand := make([]byte, SecretLen)
	if _, err := rand.Read(secretRand); err != nil {
		return Config{}, fmt.Errorf("secret rand: %w", err)
	}

	secret := base58.Encode(secretRand)[:SecretLen]

	secretRouter, err := box.SealAnonymous(nil, []byte(secret), &routerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("secret router seal: %w", err)
	}

	secretShuffler, err := box.SealAnonymous(nil, []byte(secret), &shufflerPub, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("secret shuffler seal: %w", err)
	}

	return Config{
		secret:         secret,
		name:           g.name,
		host:           g.host,
		port:           g.port,
		routerSecret:   secretRouter,
		shufflerSecret: secretShuffler,
	}, nil
}
