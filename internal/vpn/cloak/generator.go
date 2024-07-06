package cloak

import (
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func Generate(brigade *storage.Brigade, user *storage.User, nacl utils.NaCl, epData map[string]string) (Config, error) {
	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return Config{}, fmt.Errorf("endpoint pub: %w", err)
	}

	bypassID := uuid.New()

	bypassenc, err := nacl.Seal(bypassID[:])
	if err != nil {
		return Config{}, fmt.Errorf("encrypt: %w", err)
	}

	epData["cloak-uid"] = bypassenc.Router.Base64()
	user.CloakByPassUIDRouterEnc = bypassenc.Router.Base64()
	user.CloakByPassUIDShufflerEnc = bypassenc.Shuffler.Base64()

	cfg := NewConfig(
		storage.GetEndpointHost(brigade, user),
		epPub.String(),
		base64.StdEncoding.EncodeToString(bypassID[:]),
		"chrome",
		"openvpn",
		brigade.CloakFakeDomain,
	)

	return cfg, nil
}
