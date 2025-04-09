package outline

import (
	"fmt"

	"github.com/vpngen/keydesk/internal/vpn/ss"
)

const Prefix = "%16%03%01%00%C2%A8%01%01"

func NewFromSS(name string, cfg ss.Config) (string, error) {
	return fmt.Sprintf("%s/?outline=1&prefix=%s",
		cfg.GetConnString(),
		Prefix,
		// strings.ReplaceAll(url.QueryEscape(name), "+", "%20"), // remove from socket
	), nil
}
