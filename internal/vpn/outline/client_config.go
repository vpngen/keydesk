package outline

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/vpngen/keydesk/internal/vpn/ss"
)

func NewFromSS(name string, cfg ss.Config) (string, error) {
	return fmt.Sprintf("%s#%s",
		cfg.GetConnString(),
		strings.ReplaceAll(url.QueryEscape(name), "+", "%20"),
	), nil
}
