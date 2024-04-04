package wg

import (
	"github.com/vpngen/keydesk/internal/vpn"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

type Config struct {
	pub, priv, psk, epPub wgtypes.Key
	ip4, ip6, dns4, dns6  netip.Addr
	host, userName        string
	port                  uint16
}

func (c Config) Protocol() string {
	return vpn.WG
}
