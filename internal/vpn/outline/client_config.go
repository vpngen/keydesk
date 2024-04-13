package outline

import "github.com/vpngen/keydesk/internal/vpn/endpoint"

func (c Config) GetClientConfig(_ endpoint.APIResponse) (any, error) {
	return c.GetAccessKey(c.name, c.host, c.port), nil
}
