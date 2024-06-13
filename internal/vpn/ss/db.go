package ss

import (
	"encoding/base64"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func (c Config) Store(user *storage.User) error {
	user.OutlineSecretRouterEnc = base64.StdEncoding.EncodeToString(c.routerSecret)
	user.OutlineSecretShufflerEnc = base64.StdEncoding.EncodeToString(c.shufflerSecret)
	return nil
}
