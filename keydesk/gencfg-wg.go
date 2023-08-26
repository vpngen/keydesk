package keydesk

import (
	"encoding/base64"
	"fmt"
	"net/netip"

	"github.com/vpngen/keydesk/keydesk/storage"
)

func GenConfWireguard(u *storage.UserConfig, wgPriv, wgPSK []byte) string {
	tmpl := `[Interface]
Address = %s
PrivateKey = %s
DNS = %s

[Peer]
Endpoint = %s:%d
PublicKey = %s
PresharedKey = %s
AllowedIPs = 0.0.0.0/0,::/0
`

	endpointHostString := u.EndpointDomain
	if endpointHostString == "" {
		endpointHostString = u.EndpointIPv4.String()
	}

	wgconf := fmt.Sprintf(tmpl,
		netip.PrefixFrom(u.IPv4, 32).String()+","+netip.PrefixFrom(u.IPv6, 128).String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv),
		u.DNSv4.String()+","+u.DNSv6.String(),
		endpointHostString, u.EndPointPort,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(u.EndpointWgPublic),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPSK),
	)

	return wgconf
}
