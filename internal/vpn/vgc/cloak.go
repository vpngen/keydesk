package vgc

type (
	Cloak struct {
		RemoteHost string
		RemotePort int
		UID        string
		PublicKey  string
		ProxyBook  ProxyBook
	}
	ProxyBook struct {
		Shadowsocks Shadowsocks `json:"shadowsocks"`
	}
)

func NewCloak(remoteHost, uid, publicKey string, remotePort int, proxyBook ProxyBook) Cloak {
	return Cloak{RemoteHost: remoteHost, RemotePort: remotePort, UID: uid, PublicKey: publicKey, ProxyBook: proxyBook}
}

func NewCloakDefault(remoteHost, uid, publicKey string, proxyBook ProxyBook) Cloak {
	return NewCloak(remoteHost, uid, publicKey, 443, proxyBook)
}
