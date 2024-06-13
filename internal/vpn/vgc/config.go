package vgc

type Config struct {
	Config      config      `json:"config"`
	Wireguard   Wireguard   `json:"wireguard"`
	Cloak       Cloak       `json:"cloak"`
	Shadowsocks Shadowsocks `json:"shadowsocks"`
}

func New(name string, version, extended int, wg Wireguard, ck Cloak, ss Shadowsocks) Config {
	return Config{
		Config:      config{version, name, extended},
		Wireguard:   wg,
		Cloak:       ck,
		Shadowsocks: ss,
	}
}

func NewV1(name string, wg Wireguard, ck Cloak, ss Shadowsocks) Config {
	return Config{
		Config:      config{1, name, 1},
		Wireguard:   wg,
		Cloak:       ck,
		Shadowsocks: ss,
	}
}

type config struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	Extended int    `json:"extended"`
}
