package wg

import (
	"encoding/base64"
	"strings"
)

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"wg-public-key": base64.StdEncoding.EncodeToString(c.epPub[:]),
		"wg-psk-key":    base64.StdEncoding.EncodeToString(c.routerPSK[:]),
		"allowed-ips":   strings.Join([]string{c.ip4.String(), c.ip6.String()}, ","),
	}, nil
}
