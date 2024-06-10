package ipsec

import (
	"encoding/base64"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func (c Config) Store(user *storage.User) error {
	user.IPSecUsernameRouterEnc = base64.StdEncoding.EncodeToString(c.routerUser)
	user.IPSecUsernameShufflerEnc = base64.StdEncoding.EncodeToString(c.shufflerUser)
	user.IPSecPasswordRouterEnc = base64.StdEncoding.EncodeToString(c.routerPass)
	user.IPSecPasswordShufflerEnc = base64.StdEncoding.EncodeToString(c.shufflerPass)
	return nil
}
