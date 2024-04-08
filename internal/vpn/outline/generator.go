package outline

import (
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/internal/vpn"
)

type Generator struct {
	name, host string
	port       uint16
}

func NewGenerator(name string, host string, port uint16) Generator {
	return Generator{name: name, host: host, port: port}
}

func (g Generator) Generate() (vpn.Config, error) {
	secretRand := make([]byte, SecretLen)
	if _, err := rand.Read(secretRand); err != nil {
		return Config{}, fmt.Errorf("secret rand: %w", err)
	}
	return Config{
		secret: base58.Encode(secretRand),
		name:   g.name,
		host:   g.host,
		port:   g.port,
	}, nil
}
