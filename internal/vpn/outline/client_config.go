package outline

func (c Config) GetClientConfig() (any, error) {
	return c.GetAccessKey(c.name, c.host, c.port), nil
}
