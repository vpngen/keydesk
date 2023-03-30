package storage

import (
	"fmt"
	"os"

	"github.com/vpngen/keydesk/vpnapi"
)

// CreateBrigade - create brigade config.
func (db *BrigadeStorage) CreateBrigade(config *BrigadeConfig, wgPub, wgRouterPriv, wgShufflerPriv []byte) error {
	f, data, err := db.openWithoutReading(config.BrigadeID)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	db.calculatedAddrPort = vpnapi.CalcAPIAddrPort(config.EndpointIPv4)
	fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", db.calculatedAddrPort)

	switch {
	case db.APIAddrPort.Addr().IsValid() && db.APIAddrPort.Addr().IsUnspecified():
		db.actualAddrPort = db.calculatedAddrPort
	default:
		db.actualAddrPort = db.APIAddrPort
		if db.actualAddrPort.IsValid() {
			fmt.Fprintf(os.Stderr, "API endpoint: %s\n", db.actualAddrPort)
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
	err = vpnapi.WgAdd(db.actualAddrPort, db.actualAddrPort, data.WgPrivateRouterEnc, config.EndpointIPv4, config.IPv4CGNAT, config.IPv6ULA)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	err = commitBrigade(f, data)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// DestroyBrigade - remove brigade.
func (db *BrigadeStorage) DestroyBrigade() error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	// if we catch a slowdown problems we need organize queue
	err = vpnapi.WgDel(db.actualAddrPort, db.calculatedAddrPort, data.WgPrivateRouterEnc)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	return nil
}
