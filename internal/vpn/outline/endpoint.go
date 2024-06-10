package outline

import "encoding/base64"

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"outline-ss-password": base64.StdEncoding.EncodeToString(c.routerSecret),
	}, nil
}
