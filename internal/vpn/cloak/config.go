package cloak

import (
	"strconv"

	"github.com/vpngen/keydesk/internal/vpn/ss"
)

const (
	defaultEncryptionMethod = "aes-gcm"
	defaultStreamTimeout    = 300 // seconds
	defaultNumConn          = 1
	defaultRemotePort       = "443"
	defaultTransport        = "direct"
)

type Config struct {
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

func (c Config) GetVGC(book ProxyBook) (*VGC, error) {
	port, err := strconv.Atoi(c.RemotePort)
	if err != nil {
		return nil, err
	}
	return &VGC{
		RemoteHost: c.RemoteHost,
		RemotePort: port,
		UID:        c.UID,
		PublicKey:  c.PublicKey,
		ProxyBook:  book,
	}, nil
}

func NewConfig(domain, pubKey, uid, browser, proxyMethod, fakeDomain string) Config {
	return Config{
		PublicKey:        pubKey,
		RemoteHost:       domain,
		UID:              uid,
		ProxyMethod:      proxyMethod,             // openvpn
		BrowserSig:       browser,                 // chrome
		EncryptionMethod: defaultEncryptionMethod, // aes-gcm
		NumConn:          defaultNumConn,          // 1
		RemotePort:       defaultRemotePort,       // 443
		ServerName:       fakeDomain,              // yandex.com
		StreamTimeout:    defaultStreamTimeout,    // 300 seconds
		Transport:        defaultTransport,        // direct
	}
}

type (
	VGC struct {
		RemoteHost string    `json:"RemoteHost"`
		RemotePort int       `json:"RemotePort"`
		UID        string    `json:"UID"`
		PublicKey  string    `json:"PublicKey"`
		ProxyBook  ProxyBook `json:"ProxyBook"`
		ServerName string    `json:"ServerName"`
	}
	ProxyBook struct {
		Shadowsocks ss.Config `json:"shadowsocks"`
	}
)

func NewCloak(remoteHost, uid, publicKey string, remotePort int, serverName string, proxyBook ProxyBook) *VGC {
	return &VGC{RemoteHost: remoteHost, RemotePort: remotePort, UID: uid, PublicKey: publicKey, ServerName: serverName, ProxyBook: proxyBook}
}

func NewCloakDefault(remoteHost, uid, publicKey string, serverName string, proxyBook ProxyBook) *VGC {
	return NewCloak(remoteHost, uid, publicKey, 443, serverName, proxyBook)
}
