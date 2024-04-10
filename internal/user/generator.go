package user

import (
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/ipsec"
	"github.com/vpngen/keydesk/internal/vpn/outline"
	"github.com/vpngen/keydesk/internal/vpn/ovc"
	"github.com/vpngen/keydesk/internal/vpn/wg"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

func newGenerator(
	protocol string,
	brigade storage.Brigade,
	user storage.User,
	wgPriv, wgPub wgtypes.Key,
) (g vpn.Generator, err error) {
	switch protocol {
	default:
		err = fmt.Errorf("unsupported VPN protocol: %s", protocol)
	case vpn.WG:
		g, err = newWGGenerator(brigade, wgPriv, wgPub, user.IPv4Addr, user.IPv6Addr, user.Name)
	case vpn.IPSec:
		g = newIPSecGenerator(brigade)
	case vpn.Outline:
		g = newOutlineGenerator(brigade, user.Name)
	case vpn.OVC:
		g, err = newOVCGenerator(brigade, user.Name, brigade.EndpointIPv4)
	}
	return
}

func newWGGenerator(brigade storage.Brigade, wgPriv, wgPub wgtypes.Key, ip4, ip6 netip.Addr, userName string) (vpn.Generator, error) {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return nil, fmt.Errorf("bad ep wg publick key: %w", err)
	}
	return wg.NewGenerator(
		wgPriv,
		wgPub,
		epPub,
		ip4,
		ip6,
		brigade.DNSv4,
		brigade.DNSv6,
		host,
		userName,
		brigade.EndpointPort,
	), nil
}

func newOutlineGenerator(brigade storage.Brigade, userName string) vpn.Generator {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	return outline.NewGenerator(userName, host, brigade.OutlinePort)
}

func newIPSecGenerator(brigade storage.Brigade) vpn.Generator {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	return ipsec.NewGenerator(brigade.IPSecPSK, host)
}

func newOVCGenerator(brigade storage.Brigade, name string, ep4 netip.Addr) (vpn.Generator, error) {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	caPem, err := kdlib.Unbase64Ungzip(brigade.OvCACertPemGzipBase64)
	if err != nil {
		return nil, fmt.Errorf("unbase64 ca: %w", err)
	}
	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return nil, fmt.Errorf("bad ep wg publick key: %w", err)
	}
	return ovc.NewGenerator(
		host,
		name,
		brigade.CloakFakeDomain,
		string(caPem),
		ep4,
		epPub,
	), nil
}
