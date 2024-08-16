package ss

import (
	"encoding/base64"
	"fmt"
)

const (
	SecretLen  = 96
	encryption = "chacha20-ietf-poly1305"
)

type Config struct {
	Host     string `json:"host,omitempty"`
	Port     uint16 `json:"port,omitempty"`
	Cipher   string `json:"cipher"`
	Password string `json:"password"`
}

func NewSS(host, cipher, password string, port uint16) Config {
	return Config{Host: host, Port: port, Cipher: cipher, Password: password}
}

func NewSSProxyBook(cipher, password string) Config {
	return Config{Cipher: cipher, Password: password}
}

func (c Config) GetConnString() string {
	return "ss://" + base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(
		fmt.Appendf([]byte{}, "%s:%s", encryption, c.Password),
	) +
		"@" + fmt.Sprintf("%s:%d", c.Host, c.Port)
	// return fmt.Sprintf("%s:%s@%s:%d", encryption, c.Password, c.Host, c.Port)
}
