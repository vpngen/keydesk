package outline

import (
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"net/url"
)

const (
	SecretLen  = 96
	encryption = "chacha20-ietf-poly1305"
)

type Config struct {
	secret, name, host           string
	port                         uint16
	routerSecret, shufflerSecret []byte
}

func (c Config) Protocol() string {
	return vpn.Outline
}

func (c Config) getConnString(host string, port uint16) string {
	return fmt.Sprintf("%s:%s@%s:%d", encryption, c.secret, host, port)
}

func (c Config) GetAccessKey(name, host string, port uint16) string {
	return fmt.Sprintf(
		"ss://%s#%s",
		base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(c.getConnString(host, port))),
		url.QueryEscape(name),
	)
}
