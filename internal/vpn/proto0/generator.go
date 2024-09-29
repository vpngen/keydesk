package proto0

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
)

func Generate(brigade *storage.Brigade, user *storage.User, nacl utils.NaCl, epData map[string]string) (*Config, error) {
	longID := uuid.New().String()
	shortID := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	secret := shortID + "-" + strings.ReplaceAll(longID, "-", "")

	secretenc, err := nacl.Seal([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	epData["p0-id"] = secretenc.Router.Base64()
	user.Proto0SecretRouterEnc = secretenc.Router.Base64()
	user.Proto0SecretShufflerEnc = secretenc.Shuffler.Base64()

	cfg := NewProto0(brigade.WgPublicKey, longID, shortID, storage.GetEndpointHost(brigade, user), brigade.Proto0FakeDomain, brigade.Proto0Port)

	return cfg, nil
}
