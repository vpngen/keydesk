package wg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/kdlib"
	"net/netip"
	"strings"
	"text/template"
)

const templateString = `[Interface]
Address = {{ .Address }}
PrivateKey = {{ .PrivateKey }}
DNS = {{ .DNS }}

[Peer]
Endpoint = {{ .EndpointHost }}:{{ .EndpointPort }}
PublicKey = {{ .EndpointPub }}
PresharedKey = {{ .PSK }}
AllowedIPs = 0.0.0.0/0,::/0
`

var tmpl = template.Must(template.New("wireguard.conf").Parse(templateString))

type clientConfig struct {
	Address      string
	PrivateKey   string
	DNS          string
	EndpointHost string
	EndpointPort uint16
	EndpointPub  string
	PSK          string
}

func (c Config) GetClientConfig() (any, error) {
	buf, err := c.renderClientConfig()
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	wgName := kdlib.AssembleWgStyleTunName(c.userName)

	return vpn.FileConfig{
		Content:    buf.String(),
		FileName:   wgName + ".conf",
		ConfigName: wgName,
	}, nil
}

func (c Config) renderClientConfig() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	dot := clientConfig{
		Address: strings.Join([]string{
			netip.PrefixFrom(c.ip4, 32).String(),
			netip.PrefixFrom(c.ip6, 128).String(),
		}, ","),
		PrivateKey:   base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(c.priv[:]),
		DNS:          strings.Join([]string{c.dns4.String(), c.dns6.String()}, ","),
		EndpointHost: c.host,
		EndpointPort: c.port,
		EndpointPub:  base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(c.epPub[:]),
		PSK:          base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(c.psk[:]),
	}

	if err := tmpl.Execute(buf, dot); err != nil {
		return nil, err
	}

	return buf, nil
}
