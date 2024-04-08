package outline

func (c Config) GetEndpointParams() (map[string]string, error) {
	return map[string]string{
		"outline-ss-password": c.secret,
	}, nil
}
