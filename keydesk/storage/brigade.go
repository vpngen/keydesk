package storage

import (
	"fmt"

	"github.com/vpngen/keydesk/epapi"
)

// CreateBrigade - create brigade config.
func (db *BrigadeStorage) CreateBrigade(config *BrigadeConfig, wgPub, wgRouterPriv, wgShufflerPriv []byte) error {
	dt, data, stat, err := db.openWithoutReading(config.BrigadeID)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	addr := db.APIAddrPort
	if addr.Addr().IsValid() && addr.Addr().IsUnspecified() {
		addr = epapi.CalcAPIAddrPort(config.EndpointIPv4)
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
	err = epapi.WgAdd(addr, data.WgPrivateRouterEnc, config.EndpointIPv4, config.IPv4CGNAT, config.IPv6ULA)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	err = dt.save(data, stat)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// DestroyBrigade - remove brigade.
func (db *BrigadeStorage) DestroyBrigade() error {
	dt, data, stat, addr, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	// if we catch a slowdown problems we need organize queue
	err = epapi.WgDel(addr, data.WgPrivateRouterEnc)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	data = &Brigade{}

	dt.save(data, stat)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
