package amnezia

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

func Unmarshal(reader io.Reader, config *Config) error {
	dec, err := DecodeToJSON(reader)
	if err != nil {
		return fmt.Errorf("decode amnezia: %w", err)
	}
	defer dec.Close()

	jsonDec := json.NewDecoder(dec)

	if err = jsonDec.Decode(config); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	return nil
}

func DecodeToJSON(reader io.Reader) (io.ReadCloser, error) {
	prefix := make([]byte, 6)
	if _, err := reader.Read(prefix); err != nil {
		return nil, fmt.Errorf("read prefix: %w", err)
	}

	if !bytes.Equal(prefix, []byte("vpn://")) {
		return nil, fmt.Errorf("unexpected prefix: %q", prefix)
	}

	b64dec := base64.NewDecoder(base64.URLEncoding.WithPadding(base64.NoPadding), reader)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, b64dec); err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	zdec, err := zlib.NewReader(bytes.NewReader(buf.Bytes()[4:]))
	if err != nil {
		return nil, fmt.Errorf("new zlib reader: %w", err)
	}

	return zdec, nil
}

func NewConfig(hostname, vpnName, dns1, dns2 string) Config {
	return Config{
		Description: vpnName,
		HostName:    hostname,
		DNS1:        dns1,
		DNS2:        dns2,
	}
}
