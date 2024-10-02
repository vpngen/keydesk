package outline

import (
	"fmt"

	"github.com/vpngen/keydesk/internal/vpn/ss"
	"github.com/vpngen/keydesk/keydesk"
)

func NewFromSS(name string, cfg ss.Config) (string, error) {
	return fmt.Sprintf("%s/?outline=1&prefix=%s",
		cfg.GetConnString(),
		keydesk.OutlinePrefix,
		// strings.ReplaceAll(url.QueryEscape(name), "+", "%20"), // remove from socket
	), nil
}
