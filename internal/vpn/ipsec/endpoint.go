package ipsec

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"l2tp-username": c.username,
		"l2tp-password": c.password,
	}, nil
}
