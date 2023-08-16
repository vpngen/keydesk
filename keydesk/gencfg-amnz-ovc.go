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

const (
	defaultOvcBrowserSig = "chrome"
)

func NewCloackConfig(domain, pubKey, uid, browser, fakeDomain string) CloakConfig {
	return CloakConfig{
		BrowserSig:       browser, // chrome
		EncryptionMethod: "aes-gcm",
		NumConn:          1,
		ProxyMethod:      "openvpn",
		PublicKey:        pubKey,
		RemoteHost:       domain,
		RemotePort:       "443",
		ServerName:       fakeDomain, // yandex.com
		StreamTimeout:    300,
		Transport:        "direct",
		UID:              uid,
	}
}

type AmneziaCloakConfig struct {
	LastConfig string `json:"last_config"`
	Port       string `json:"port"`            // 443
	Transport  string `json:"transport_proto"` // tcp
}

func newAmneziaCloakConfig(cfg string) AmneziaCloakConfig {
	return AmneziaCloakConfig{
		LastConfig: cfg,
		Port:       "443",
		Transport:  "tcp",
	}
}

type AmneziaOpenVPNConfig struct {
	LastConfig string `json:"last_config"`
}

type AmneziaOpenVPNConfigJson struct {
	Config string `json:"config"`
}

func newAmneziaOpenVPNConfig(cfg string) AmneziaOpenVPNConfig {
	return AmneziaOpenVPNConfig{
		LastConfig: cfg,
	}
}

type AmneziaShadowSocksConfig struct {
	LastConfig string `json:"last_config"`
}

func newAmneziaShadowSocksConfig(cfg string) AmneziaShadowSocksConfig {
	return AmneziaShadowSocksConfig{
		LastConfig: cfg,
	}
}

type AmneziaContainer struct {
	Container   string                   `json:"container"`
	Cloak       AmneziaCloakConfig       `json:"cloak"`
	OpenVPN     AmneziaOpenVPNConfig     `json:"openvpn"`
	ShadowSocks AmneziaShadowSocksConfig `json:"shadowsocks"`
}

func newAmneziaContainer(cloak, openvpn, shadowsocks string) AmneziaContainer {
	return AmneziaContainer{
		Container:   "amnezia-openvpn-cloak",
		Cloak:       newAmneziaCloakConfig(cloak),
		OpenVPN:     newAmneziaOpenVPNConfig(openvpn),
		ShadowSocks: newAmneziaShadowSocksConfig(shadowsocks),
	}
}

type AmneziaConfig struct {
	Containers       []AmneziaContainer `json:"containers"`
	DefaultContainer string             `json:"defaultContainer"` // amnezia-openvpn-cloak
	Description      string             `json:"description"`      // VPN Generator
	DNS1             string             `json:"dns1,omitempty"`   //
	DNS2             string             `json:"dns2,omitempty"`   //
	HostName         string             `json:"hostName"`         // ${EXT_IP}
}

func NewAmneziaConfig(hostname, vpnName, cloak, openvpn, dns string) AmneziaConfig {
	a := AmneziaConfig{
		Containers: []AmneziaContainer{
			newAmneziaContainer(cloak, openvpn, "{}"),
		},
		DefaultContainer: "amnezia-openvpn-cloak",
		Description:      vpnName,
		HostName:         hostname,
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

func NewOpenVPNConfig(dns, ip, ca, cert, key string) string {
	return fmt.Sprintf(OpenVPNConfigTemplate, dns, ip, ca, cert, key)
}

func GenConfAmneziaOpenVPNoverCloak(u *storage.UserConfig, ovcKeyPriv string) (string, error) {
	endpointHostString := u.EndpointDomain
	if endpointHostString == "" {
		endpointHostString = u.EndpointIPv4.String()
	}

	cloakConfig := NewCloackConfig(
		endpointHostString,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(u.EndpointWgPublic),
		u.CloakBypassUID,
		defaultOvcBrowserSig,
		GetRandomSite(),
	)

	cloakConfigString, err := json.Marshal(cloakConfig)
	if err != nil {
		return "", fmt.Errorf("marshal cloak config: %w", err)
	}

	openvpnConfig := NewOpenVPNConfig(
		u.DNSv4.String(), //+","+u.DNSv6.String(),
		u.EndpointIPv4.String(),
		u.OvCACertPem,
		u.OvClientCertPem,
		ovcKeyPriv,
	)

	openvpnConfigConfig, err := json.Marshal(AmneziaOpenVPNConfigJson{
		Config: openvpnConfig,
	})
	if err != nil {
		return "", fmt.Errorf("marshal openvpn config: %w", err)
	}

	amneziaConfig := NewAmneziaConfig(
		endpointHostString,
		u.Name,
		string(cloakConfigString),
		string(openvpnConfigConfig),
		u.DNSv4.String(), //+","+u.DNSv6.String(),
	)

	amneziaConfigString, err := json.Marshal(amneziaConfig)
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
	if _, err := io.Copy(gzw, bytes.NewReader(amneziaConfigString)); err != nil {
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
