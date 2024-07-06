package outline

import (
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn/ss"
	"net/url"
)

func NewFromSS(name string, cfg ss.Config) (string, error) {
	return fmt.Sprintf(
		"ss://%s#%s",
		base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(cfg.GetConnString())),
		url.QueryEscape(name),
	), nil
}
