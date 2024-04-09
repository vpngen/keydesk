package ipsec

import "encoding/base64"

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"l2tp-username": base64.StdEncoding.EncodeToString(c.routerUser),
		"l2tp-password": base64.StdEncoding.EncodeToString(c.routerPass),
	}, nil
}
