package ss

import (
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
)

func Generate(brigade *storage.Brigade, user *storage.User, nacl utils.NaCl, epData map[string]string) (Config, error) {
	secretRand := make([]byte, SecretLen)
	if _, err := rand.Read(secretRand); err != nil {
		return Config{}, fmt.Errorf("secret rand: %w", err)
	}

	secret := base58.Encode(secretRand)[:SecretLen]
	secretenc, err := nacl.Seal([]byte(secret))
	if err != nil {
		return Config{}, fmt.Errorf("encrypt: %w", err)
	}

	epData["outline-ss-password"] = secretenc.Router.Base64()
	user.OutlineSecretRouterEnc = secretenc.Router.Base64()
	user.OutlineSecretShufflerEnc = secretenc.Shuffler.Base64()

	cfg := Config{
		Host:     storage.GetEndpointHost(brigade, user),
		Port:     brigade.OutlinePort,
		Cipher:   "chacha20-ietf-poly1305",
		Password: secret,
	}

	return cfg, nil
}
