package ovc

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

func (c Config) SaveToUser(user *storage.User, router, shuffler [32]byte) error {
	enc, err := c.encrypt(router, shuffler)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	csr, err := c.csrPemGzBase64()
	if err != nil {
		return fmt.Errorf("csrPemGzBase64: %w", err)
	}
	user.OvCSRGzipBase64 = string(csr)
	user.CloakByPassUIDRouterEnc = base64.StdEncoding.EncodeToString(enc.routerBypass)
	user.CloakByPassUIDShufflerEnc = base64.StdEncoding.EncodeToString(enc.shufflerBypass)
	return nil
}

func (c Config) encrypt(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (EncryptedConfig, error) {
	routerBypass, err := box.SealAnonymous(nil, c.bypass[:], &routerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("cloakBypassUID router seal: %w", err)
	}

	shufflerBypass, err := box.SealAnonymous(nil, c.bypass[:], &shufflerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("cloakBypassUID shuffler seal: %w", err)
	}

	return EncryptedConfig{
		routerBypass:   routerBypass,
		shufflerBypass: shufflerBypass,
	}, nil
}
