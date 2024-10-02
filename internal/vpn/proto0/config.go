package proto0

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	PublicKey []byte `json:"publicKey,omitempty"`
	ID        string `json:"id,omitempty"`
	ShortID   string `json:"shortId,omitempty"`

	Address string `json:"address,omitempty"`
	Port    uint16 `json:"port,omitempty"`

	ServerName string `json:"serverName,omitempty"`
}

func NewProto0(pubkey []byte, longID, shortID string, host string, sn string, port uint16) *Config {
	return &Config{
		PublicKey: pubkey,
		ID:        longID,
		ShortID:   shortID,

		Address: host,
		Port:    port,

		ServerName: sn,
	}
}

func (c Config) GetConnString(name string) string {
	conf := "\u0076\u006C\u0065\u0073\u0073\u003A\u002F\u002F" + c.ID +
		fmt.Sprintf("@%s:%d?", c.Address, c.Port) +
		"\u0073\u0065\u0063\u0075\u0072\u0069\u0074\u0079\u003D\u0072\u0065\u0061\u006C\u0069\u0074\u0079" +
		"\u0026\u0065\u006E\u0063\u0072\u0079\u0070\u0074\u0069\u006F\u006E\u003D\u006E\u006F\u006E\u0065" + "\u0026\u0070\u0062\u006B\u003D" +
		base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(c.PublicKey) +
		"\u0026\u0068\u0065\u0061\u0064\u0065\u0072\u0054\u0079\u0070\u0065\u003D\u006E\u006F\u006E\u0065" +
		"\u0026\u0066\u0070\u003D\u0063\u0068\u0072\u006F\u006D\u0065\u0026\u0074\u0079\u0070\u0065\u003D" +
		"\u0074\u0063\u0070\u0026\u0066\u006C\u006F\u0077\u003D\u0078\u0074\u006C\u0073\u002D\u0072\u0070\u0072\u0078\u002D\u0076\u0069\u0073\u0069\u006F\u006E" +
		"\u0026\u0073\u006E\u0069\u003D" + c.ServerName +
		"\u0026\u0073\u0069\u0064\u003D" + c.ShortID

	if name != "" {
		conf += "#" + strings.ReplaceAll(url.QueryEscape(name), "+", "%20")
	}

	return conf
}
