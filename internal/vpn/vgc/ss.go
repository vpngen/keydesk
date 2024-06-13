package vgc

type Shadowsocks struct {
	Host     string `json:"host,omitempty"`
	Port     uint16 `json:"port,omitempty"`
	Cipher   string `json:"cipher"`
	Password string `json:"password"`
}

func NewSS(host, cipher, password string, port uint16) Shadowsocks {
	return Shadowsocks{Host: host, Port: port, Cipher: cipher, Password: password}
}

func NewSSProxyBook(cipher, password string) Shadowsocks {
	return Shadowsocks{Cipher: cipher, Password: password}
}
