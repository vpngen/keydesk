package ovc

import (
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func (c Config) Store(user *storage.User) error {
	csr, err := c.csrPemGzBase64()
	if err != nil {
		return fmt.Errorf("csrPemGzBase64: %w", err)
	}
	user.OvCSRGzipBase64 = string(csr)
	user.CloakByPassUIDRouterEnc = base64.StdEncoding.EncodeToString(c.routerBypass)
	user.CloakByPassUIDShufflerEnc = base64.StdEncoding.EncodeToString(c.shufflerBypass)
	return nil
}
