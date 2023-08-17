package keydesk

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vpngen/keydesk/keydesk/storage"
)

const (
	defaultCloakBrowserSig       = "chrome"
	defaultCloakEncryptionMethod = "aes-gcm"
	defaultCloakStreamTimeout    = 300 // seconds
	defaultCloakNumConn          = 1
	defaultCloakRemotePort       = "443"
	defaultCloakTransport        = "direct"
	cloakProxyMethodOpenVPN      = "openvpn"
)

type CloakConfig struct {
	BrowserSig       string `json:"BrowserSig"`
	EncryptionMethod string `json:"EncryptionMethod"`
	NumConn          int    `json:"NumConn"`
	ProxyMethod      string `json:"ProxyMethod"`
	PublicKey        string `json:"PublicKey"`
	RemoteHost       string `json:"RemoteHost"`
	RemotePort       string `json:"RemotePort"`
	ServerName       string `json:"ServerName"`
	StreamTimeout    int    `json:"StreamTimeout"`
	Transport        string `json:"Transport"`
	UID              string `json:"UID"`
}

func NewCloackConfig(domain, pubKey, uid, browser, proxyMethod, fakeDomain string) (string, error) {
	conf, err := json.Marshal(&CloakConfig{
		PublicKey:        pubKey,
		RemoteHost:       domain,
		UID:              uid,
		ProxyMethod:      proxyMethod,                  // openvpn
		BrowserSig:       browser,                      // chrome
		EncryptionMethod: defaultCloakEncryptionMethod, // aes-gcm
		NumConn:          defaultCloakNumConn,          // 1
		RemotePort:       defaultCloakRemotePort,       // 443
		ServerName:       fakeDomain,                   // yandex.com
		StreamTimeout:    defaultCloakStreamTimeout,    // 300 seconds
		Transport:        defaultCloakTransport,        // direct
	})
	if err != nil {
		return "", fmt.Errorf("marshal cloak config: %w", err)
	}

	return string(conf), nil
}

type AmneziaConfigInnerJson struct {
	Config string `json:"config"`
}

const OpenVPNConfigTemplate = `client
dev tun
proto tcp
resolv-retry infinite
nobind
persist-key
persist-tun
cipher AES-256-GCM
auth SHA512
verb 3
tls-client
tls-version-min 1.2
key-direction 1
remote-cert-tls server
redirect-gateway def1 bypass-dhcp

dhcp-option DNS %s
block-outside-dns

route %s 255.255.255.255 net_gateway
remote 127.0.0.1 1194

<ca>
%s
</ca>
<cert>
%s
</cert>
<key>
%s
</key>`

func NewOpenVPNConfigJson(dns, ip, ca, cert, key string) (string, error) {
	ov := fmt.Sprintf(OpenVPNConfigTemplate, dns, ip, ca, cert, key)

	conf, err := json.Marshal(&AmneziaConfigInnerJson{
		Config: ov,
	})
	if err != nil {
		return "", fmt.Errorf("marshal openvpn config: %w", err)
	}

	return string(conf), nil
}

func NewWireguardConfigJson(wg string) (string, error) {
	conf, err := json.Marshal(&AmneziaConfigInnerJson{
		Config: wg,
	})
	if err != nil {
		return "", fmt.Errorf("marshal wireguard config: %w", err)
	}

	return string(conf), nil
}

type AmneziaCloakConfig struct {
	LastConfig string `json:"last_config"`
	Port       string `json:"port"`            // 443
	Transport  string `json:"transport_proto"` // tcp
}

type AmneziaOpenVPNConfig struct {
	LastConfig string `json:"last_config"`
}

type AmneziaShadowSocksConfig struct {
	LastConfig string `json:"last_config"`
}

type AmneziaWireguardConfig struct {
	LastConfig string `json:"last_config"`
}

type AmneziaContainer struct {
	Container          string                    `json:"container"`
	Cloak              *AmneziaCloakConfig       `json:"cloak,omitempty"`
	OpenVPN            *AmneziaOpenVPNConfig     `json:"openvpn,omitempty"`
	ShadowSocks        *AmneziaShadowSocksConfig `json:"shadowsocks,omitempty"`
	Wireguard          *AmneziaWireguardConfig   `json:"wireguard,omitempty"`
	IsThirdPartyConfig bool                      `json:"isThirdPartyConfig,omitempty"`
}

const (
	AmneziaContainerOpenVPNCloak = "amnezia-openvpn-cloak"
	AmneziaContainerWireguard    = "amnezia-wireguard"
	CloakPort                    = "443"
	CloakTransport               = "tcp"
)

func NewAmneziaContainerWithOvc(cloak, openvpn, shadowsocks string) *AmneziaContainer {
	return &AmneziaContainer{
		Container: AmneziaContainerOpenVPNCloak,
		Cloak: &AmneziaCloakConfig{
			LastConfig: cloak,
			Port:       CloakPort,
			Transport:  CloakTransport,
		},
		OpenVPN: &AmneziaOpenVPNConfig{
			LastConfig: openvpn,
		},
		ShadowSocks: &AmneziaShadowSocksConfig{
			LastConfig: shadowsocks,
		},
	}
}

func NewAmneziaContainerWithWg(wg string) *AmneziaContainer {
	return &AmneziaContainer{
		Container: AmneziaContainerWireguard,
		Wireguard: &AmneziaWireguardConfig{
			LastConfig: wg,
		},
		IsThirdPartyConfig: true,
	}
}

type AmneziaConfig struct {
	Containers       []*AmneziaContainer `json:"containers"`
	DefaultContainer string              `json:"defaultContainer"` // amnezia-openvpn-cloak
	Description      string              `json:"description"`      // VPN Generator
	DNS1             string              `json:"dns1,omitempty"`   //
	DNS2             string              `json:"dns2,omitempty"`   //
	HostName         string              `json:"hostName"`         // ${EXT_IP}
}

func NewAmneziaConfig(hostname, vpnName, dns string) *AmneziaConfig {
	a := &AmneziaConfig{
		Description: vpnName,
		HostName:    hostname,
	}

	if dns != "" {
		dnsList := strings.Split(dns, ",")
		if len(dnsList) > 0 {
			a.DNS1 = dnsList[0]
		}
		if len(dnsList) > 1 {
			a.DNS2 = dnsList[1]
		}
	}

	return a
}

func (ac *AmneziaConfig) AddContainer(c *AmneziaContainer) {
	ac.Containers = append(ac.Containers, c)
}

func (ac *AmneziaConfig) SetDefaultContainer(c string) {
	ac.DefaultContainer = c
}

func (ac *AmneziaConfig) Marshal() (string, error) {
	conf, err := json.Marshal(ac)
	if err != nil {
		return "", fmt.Errorf("marshal amnezia config: %w", err)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.Write([]byte("vpn://")); err != nil {
		return "", fmt.Errorf("write vpn:// magick: %w", err)
	}

	// fmt.Fprintf(os.Stderr, " ********** Amnezia config: %s\n", amneziaConfigString)

	b64w := base64.NewEncoder(base64.URLEncoding.WithPadding(base64.NoPadding), buf)

	if _, err := b64w.Write([]byte{0, 0, 0, 0xff}); err != nil {
		return "", fmt.Errorf("write magic: %w", err)
	}

	gzw := zlib.NewWriter(b64w)
	if _, err := io.Copy(gzw, bytes.NewReader(conf)); err != nil {
		return "", fmt.Errorf("compress amnezia config: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return "", fmt.Errorf("close gzip: %w", err)
	}

	if err := b64w.Close(); err != nil {
		return "", fmt.Errorf("close base64: %w", err)
	}

	return buf.String(), nil
}

func GenConfAmneziaOpenVPNoverCloak(u *storage.UserConfig, ovcKeyPriv string) (*AmneziaContainer, error) {
	endpointHostString := u.EndpointDomain
	if endpointHostString == "" {
		endpointHostString = u.EndpointIPv4.String()
	}

	cloakConfig, err := NewCloackConfig(
		endpointHostString,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(u.EndpointWgPublic),
		u.CloakBypassUID,
		defaultCloakBrowserSig,
		cloakProxyMethodOpenVPN,
		GetRandomSite(),
	)
	if err != nil {
		return nil, fmt.Errorf("marshal cloak config: %w", err)
	}

	openvpnConfig, err := NewOpenVPNConfigJson(
		u.DNSv4.String(), //+","+u.DNSv6.String(),
		u.EndpointIPv4.String(),
		u.OvCACertPem,
		u.OvClientCertPem,
		ovcKeyPriv,
	)
	if err != nil {
		return nil, fmt.Errorf("marshal openvpn config: %w", err)
	}

	return NewAmneziaContainerWithOvc(cloakConfig, openvpnConfig, "{}"), nil
}

func GenConfAmneziaWireguard(wgconf string) (*AmneziaContainer, error) {
	wgConf, err := NewWireguardConfigJson(wgconf)
	if err != nil {
		return nil, fmt.Errorf("marshal wireguard config: %w", err)
	}

	return NewAmneziaContainerWithWg(wgConf), nil
}
