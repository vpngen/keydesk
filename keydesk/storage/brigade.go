package storage

import (
	"fmt"
	"net/netip"
	"os"

	"github.com/vpngen/keydesk/vapnapi"
)

// CreateBrigade - create brigade config.
func (db *BrigadeStorage) CreateBrigade(config *BrigadeConfig, wgPub, wgRouterPriv, wgShufflerPriv []byte) error {
	var addr netip.AddrPort

	dt, data, stats, err := db.openWithoutReading(config.BrigadeID)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	calculatedAddrPort := vapnapi.CalcAPIAddrPort(config.EndpointIPv4)
	fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", calculatedAddrPort)

	switch {
	case db.APIAddrPort.Addr().IsValid() && db.APIAddrPort.Addr().IsUnspecified():
		addr = calculatedAddrPort
	default:
		addr = db.APIAddrPort
		if addr.IsValid() {
			fmt.Fprintf(os.Stderr, "API endpoint: %s\n", calculatedAddrPort)
		}
	}

	data.WgPublicKey = wgPub
	data.WgPrivateRouterEnc = wgRouterPriv
	data.WgPrivateShufflerEnc = wgShufflerPriv
	data.IPv4CGNAT = config.IPv4CGNAT
	data.IPv6ULA = config.IPv6ULA
	data.DNSv4 = config.DNSIPv4
	data.DNSv6 = config.DNSIPv6
	data.EndpointIPv4 = config.EndpointIPv4
	data.KeydeskIPv6 = config.KeydeskIPv6

	// if we catch a slowdown problems we need organize queue
	err = vapnapi.WgAdd(addr, data.WgPrivateRouterEnc, config.EndpointIPv4, config.IPv4CGNAT, config.IPv6ULA)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	err = dt.save(data, stats)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// DestroyBrigade - remove brigade.
func (db *BrigadeStorage) DestroyBrigade() error {
	dt, data, stats, addr, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	// if we catch a slowdown problems we need organize queue
	err = vapnapi.WgDel(addr, data.WgPrivateRouterEnc)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	data = &Brigade{}

	dt.save(data, stats)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
