package wg

import (
	"crypto/rand"
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

func (c Config) Store(user *storage.User, router, shuffler [naclkey.NaclBoxKeyLength]byte) error {
	routerPSK, shufflerPSK, err := c.encrypt(router, shuffler)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	user.WgPublicKey = c.pub[:]
	user.WgPSKRouterEnc = routerPSK
	user.WgPSKShufflerEnc = shufflerPSK
	return nil
}

func (c Config) encrypt(routerPub, sufflerPub [naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, error) {
	routerPsk, err := box.SealAnonymous(nil, c.psk[:], &routerPub, rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("psk router seal: %w", err)
	}

	shufflerPsk, err := box.SealAnonymous(nil, c.psk[:], &sufflerPub, rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("psk shuffler seal: %w", err)
	}

	return routerPsk, shufflerPsk, nil
}
