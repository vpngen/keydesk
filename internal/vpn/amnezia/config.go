package amnezia

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/openvpn"
)

const (
	ContainerOpenVPNCloak = "amnezia-openvpn-cloak"
	ContainerWireguard    = "amnezia-wireguard"
	CloakPort             = "443"
	CloakTransport        = "tcp"
)

type (
	CloakConfig struct {
		LastConfig string `json:"last_config"`
		Port       string `json:"port"`
		Transport  string `json:"transport_proto"`
	}

	OpenVPNConfig struct {
		LastConfig string `json:"last_config"`
	}

	ShadowSocksConfig struct {
		LastConfig string `json:"last_config"`
	}

	WireguardConfig struct {
		LastConfig string `json:"last_config"`
	}

	Container struct {
		Container          string             `json:"container"`
		Cloak              *CloakConfig       `json:"cloak,omitempty"`
		OpenVPN            *OpenVPNConfig     `json:"openvpn,omitempty"`
		ShadowSocks        *ShadowSocksConfig `json:"shadowsocks,omitempty"`
		Wireguard          *WireguardConfig   `json:"wireguard,omitempty"`
		IsThirdPartyConfig bool               `json:"isThirdPartyConfig,omitempty"`
	}

	Config struct {
		Containers       []Container `json:"containers"`
		DefaultContainer string      `json:"defaultContainer"`
		Description      string      `json:"description"`
		DNS1             string      `json:"dns1,omitempty"`
		DNS2             string      `json:"dns2,omitempty"`
		HostName         string      `json:"hostName"`
	}

	ConfigInnerJson struct {
		Config string `json:"config"`
	}
)

func (c *Config) AddContainer(container Container) {
	c.Containers = append(c.Containers, container)
}

func (c *Config) SetDefaultContainer(dc string) {
	c.DefaultContainer = dc
}

func (c *Config) Marshal() (string, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.Write([]byte("vpn://")); err != nil {
		return "", fmt.Errorf("write vpn:// magick: %w", err)
	}

	b64w := base64.NewEncoder(base64.URLEncoding.WithPadding(base64.NoPadding), buf)

	if _, err := b64w.Write([]byte{0, 0, 0, 0xff}); err != nil {
		return "", fmt.Errorf("write magic: %w", err)
	}

	gzw := zlib.NewWriter(b64w)

	enc := json.NewEncoder(gzw)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(c); err != nil {
		return "", fmt.Errorf("encode amnezia config: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return "", fmt.Errorf("close gzip: %w", err)
	}

	if err := b64w.Close(); err != nil {
		return "", fmt.Errorf("close base64: %w", err)
	}

	return buf.String(), nil
}

func NewOVCContainer(cloakCfg cloak.Config, ovpnCfg openvpn.Config) (Container, error) {
	cloakJSON := new(bytes.Buffer)
	if err := json.NewEncoder(cloakJSON).Encode(cloakCfg); err != nil {
		return Container{}, fmt.Errorf("marshal cloak config: %w", err)
	}

	ovpnCfgStr, err := ovpnCfg.Render()
	if err != nil {
		return Container{}, fmt.Errorf("render openvpn config: %w", err)
	}
	ovpnJSON := new(bytes.Buffer)
	enc := json.NewEncoder(ovpnJSON)
	enc.SetEscapeHTML(false)

	if err = enc.Encode(ConfigInnerJson{
		Config: ovpnCfgStr.String(),
	}); err != nil {
		return Container{}, fmt.Errorf("marshal openvpn config: %w", err)
	}

	return Container{
		Container: ContainerOpenVPNCloak,
		Cloak: &CloakConfig{
			LastConfig: cloakJSON.String(),
			Port:       CloakPort,
			Transport:  CloakTransport,
		},
		OpenVPN:            &OpenVPNConfig{LastConfig: ovpnJSON.String()},
		ShadowSocks:        &ShadowSocksConfig{LastConfig: "{}"},
		Wireguard:          nil,
		IsThirdPartyConfig: false,
	}, nil
}

func NewConfig(hostname, vpnName, dns1, dns2 string) Config {
	return Config{
		Description: vpnName,
		HostName:    hostname,
		DNS1:        dns1,
		DNS2:        dns2,
	}
}
