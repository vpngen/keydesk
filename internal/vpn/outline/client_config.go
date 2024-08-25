package outline

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/vpngen/keydesk/internal/vpn/ss"
	"github.com/vpngen/keydesk/keydesk"
)

func NewFromSS(name string, cfg ss.Config) (string, error) {
	return fmt.Sprintf("%s/?outline=1&prefix=%s#%s",
		cfg.GetConnString(),
		keydesk.OutlinePrefix,
		strings.ReplaceAll(url.QueryEscape(name), "+", "%20"),
	), nil
}
