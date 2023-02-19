package storage

import (
	"fmt"
	"time"

	"github.com/vpngen/keydesk/epapi"
)

// CreateBrigade - create brigade config.
func (db *BrigadeStorage) CreateBrigade(config *BrigadeConfig, wgPub, wgRouterPriv, wgShufflerPriv []byte) error {
	f, data, err := db.openWithoutReading(config.BrigadeID)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	addr := db.APIAddrPort
	if addr.Addr().IsValid() && addr.Addr().IsUnspecified() {
		addr = epapi.CalcAPIAddrPort(config.EndpointIPv4)
	}

	data = &Brigade{
		BrigadeID:            config.BrigadeID,
		CreatedAt:            time.Now(),
		WgPublicKey:          wgPub,
		WgPrivateRouterEnc:   wgRouterPriv,
		WgPrivateShufflerEnc: wgShufflerPriv,
		IPv4CGNAT:            config.IPv4CGNAT,
		IPv6ULA:              config.IPv6ULA,
		DNSv4:                config.DNSIPv4,
		DNSv6:                config.DNSIPv6,
		EndpointIPv4:         config.EndpointIPv4,
		KeydeskIPv6:          config.KeydeskIPv6,
	}

	// if we catch a slowdown problems we need organize queue
	err = epapi.WgAdd(addr, data.WgPrivateRouterEnc, config.EndpointIPv4, config.IPv4CGNAT, config.IPv6ULA)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	err = db.save(f, data)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// DestroyBrigade - remove brigade.
func (db *BrigadeStorage) DestroyBrigade() error {
	f, data, addr, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	// if we catch a slowdown problems we need organize queue
	err = epapi.WgDel(addr, data.WgPrivateRouterEnc)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	data = &Brigade{}

	db.save(f, data)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
