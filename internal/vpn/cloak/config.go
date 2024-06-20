package cloak

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
