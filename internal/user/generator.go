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
)

func newGenerator(
	protocol string,
	brigade *storage.Brigade,
	user *storage.User,
	wgPriv, wgPub wgtypes.Key,
) (g vpn.Generator, err error) {
	switch protocol {
	default:
		err = fmt.Errorf("unsupported VPN protocol: %s", protocol)
	case vpn.WG:
		g, err = newWGGenerator(brigade, user, wgPriv, wgPub)
	case vpn.IPSec:
		g = newIPSecGenerator(brigade, user)
	case vpn.Outline:
		g = newOutlineGenerator(brigade, user)
	case vpn.OVC:
		g, err = newOVCGenerator(brigade, user)
	}
	return
}

func getHost(brigade *storage.Brigade, user *storage.User) string {
	if user.EndpointDomain != "" {
		return user.EndpointDomain
	}
	if brigade.EndpointDomain != "" {
		return brigade.EndpointDomain
	}
	return brigade.EndpointIPv4.String()
}

func newWGGenerator(brigade *storage.Brigade, user *storage.User, wgPriv, wgPub wgtypes.Key) (vpn.Generator, error) {
	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return nil, fmt.Errorf("bad ep wg publick key: %w", err)
	}
	return wg.NewGenerator(
		wgPriv,
		wgPub,
		epPub,
		user.IPv4Addr,
		user.IPv6Addr,
		brigade.DNSv4,
		brigade.DNSv6,
		getHost(brigade, user),
		user.Name,
		brigade.EndpointPort,
	), nil
}

func newOutlineGenerator(brigade *storage.Brigade, user *storage.User) vpn.Generator {
	host := brigade.EndpointDomain
	if host == "" {
		host = brigade.EndpointIPv4.String()
	}
	return outline.NewGenerator(user.Name, host, brigade.OutlinePort)
}

func newIPSecGenerator(brigade *storage.Brigade, user *storage.User) vpn.Generator {
	return ipsec.NewGenerator(brigade.IPSecPSK, getHost(brigade, user))
}

func newOVCGenerator(brigade *storage.Brigade, user *storage.User) (vpn.Generator, error) {
	caPem, err := kdlib.Unbase64Ungzip(brigade.OvCACertPemGzipBase64)
	if err != nil {
		return nil, fmt.Errorf("unbase64 ca: %w", err)
	}
	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return nil, fmt.Errorf("bad ep wg publick key: %w", err)
	}
	return ovc.NewGenerator(
		getHost(brigade, user),
		user.Name,
		brigade.CloakFakeDomain,
		string(caPem),
		brigade.EndpointIPv4,
		epPub,
	), nil
}
