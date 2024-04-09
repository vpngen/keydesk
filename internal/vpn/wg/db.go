package wg

import (
	"github.com/vpngen/keydesk/keydesk/storage"
)

func (c Config) Store(user *storage.User) error {
	user.WgPublicKey = c.pub[:]
	user.WgPSKRouterEnc = c.routerPSK
	user.WgPSKShufflerEnc = c.shufflerPSK
	return nil
}
