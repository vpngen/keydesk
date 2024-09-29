package wg

import (
	"bytes"
	"fmt"
	"net/netip"
	"strings"
	"text/template"

	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type (
	RawConfig struct {
		Key          wgtypes.Key
		Address      []netip.Prefix
		DNS          []netip.Addr
		PublicKey    wgtypes.Key
		PresharedKey wgtypes.Key
		AllowedIPs   []netip.Prefix
		EndpointHost string
		EndpointPort uint16
	}
	Config2 struct {
		Interface Interface `json:"Interface"`
		Peer      Peer      `json:"Peer"`
	}
	Interface struct {
		PrivateKey string `json:"PrivateKey"`
		Address    string `json:"Address"`
		DNS        string `json:"DNS"`
	}
	Peer struct {
		PublicKey    string `json:"PublicKey"`
		PresharedKey string `json:"PresharedKey,omitempty"`
		AllowedIPs   string `json:"AllowedIPs"`
		Endpoint     string `json:"Endpoint"`
	}
)

func (c RawConfig) HandleEndpointAPIResponse(resp endpoint.APIResponse) error {
	return nil
}

func (c RawConfig) GetVGC() *Config2 {
	return &Config2{
		Interface: Interface{
			PrivateKey: c.Key.String(),
			Address:    strings.Join([]string{c.Address[0].String(), c.Address[1].String()}, ","),
			DNS:        strings.Join([]string{c.DNS[0].String(), c.DNS[1].String()}, ","),
		},
		Peer: Peer{
			PublicKey:    c.PublicKey.String(),
			PresharedKey: c.PresharedKey.String(),
			AllowedIPs:   strings.Join([]string{c.AllowedIPs[0].String(), c.AllowedIPs[1].String()}, ","),
			Endpoint:     fmt.Sprintf("%s:%d", c.EndpointHost, c.EndpointPort),
		},
	}
}

// GetAllowedIPs returns AllowedIPs string
func (c RawConfig) GetAllowedIPs() string {
	ips := make([]string, 0, len(c.AllowedIPs))
	for _, ip := range c.AllowedIPs {
		ips = append(ips, ip.String())
	}
	return strings.Join(ips, ",")
}

// GetAddresses returns Addresses string
func (c RawConfig) GetAddresses() string {
	ips := make([]string, 0, len(c.Address))
	for _, ip := range c.Address {
		ips = append(ips, ip.String())
	}
	return strings.Join(ips, ",")
}

func NewWireguard(key, addr, dns, pub, psk, ips, ep string) *Config2 {
	return &Config2{
		Interface: Interface{
			PrivateKey: key,
			Address:    addr,
			DNS:        dns,
		},
		Peer: Peer{
			PublicKey:    pub,
			PresharedKey: psk,
			AllowedIPs:   ips,
			Endpoint:     ep,
		},
	}
}

func NewWireguardAnyIP(key, addr, dns, pub, psk, ep string) *Config2 {
	return NewWireguard(key, addr, dns, pub, psk, "0.0.0.0/0,::/0", ep)
}

const templateString = `[Interface]
{{- with .Interface }}
Address = {{ .Address }}
PrivateKey = {{ .PrivateKey }}
DNS = {{ .DNS }}
{{- end }}

[Peer]
{{- with .Peer }}
Endpoint = {{ .Endpoint }}
PublicKey = {{ .PublicKey }}
PresharedKey = {{ .PresharedKey }}
AllowedIPs = {{ .AllowedIPs }}
{{- end }}
`

var tmpl = template.Must(template.New("wireguard.conf2").Parse(templateString))

func (c Config2) GetNative() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, c); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}
