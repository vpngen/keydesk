package vgc

type (
	Wireguard struct {
		Interface wgInterface `json:"Interface"`
		Peer      wgPeer      `json:"Peer"`
	}
	wgInterface struct {
		PrivateKey string `json:"PrivateKey"`
		Address    string `json:"Address"`
		DNS        string `json:"DNS"`
	}
	wgPeer struct {
		PublicKey    string `json:"PublicKey"`
		PresharedKey string `json:"PresharedKey,omitempty"`
		AllowedIPs   string `json:"AllowedIPs"`
		Endpoint     string `json:"Endpoint"`
	}
)

func NewWireguard(key, addr, dns, pub, psk, ips, ep string) Wireguard {
	return Wireguard{
		Interface: wgInterface{
			PrivateKey: key,
			Address:    addr,
			DNS:        dns,
		},
		Peer: wgPeer{
			PublicKey:    pub,
			PresharedKey: psk,
			AllowedIPs:   ips,
			Endpoint:     ep,
		},
	}
}

func NewWireguardAnyIP(key, addr, dns, pub, psk, ep string) Wireguard {
	return NewWireguard(key, addr, dns, pub, psk, "0.0.0.0/0,::/0", ep)
}
