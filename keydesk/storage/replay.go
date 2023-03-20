package storage

import (
	"fmt"
	"net/netip"

	"github.com/vpngen/keydesk/vpnapi"
)

// ReplayBrigade - create brigade config.
func (db *BrigadeStorage) ReplayBrigade(fresh, bonly, uonly bool) error {
	f, data, addr, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if fresh {
		// if we catch a slowdown problems we need organize queue
		err = vpnapi.WgDel(addr, data.WgPrivateRouterEnc)
		if err != nil {
			return fmt.Errorf("wg add: %w", err)
		}
	}

	if !uonly {
		// if we catch a slowdown problems we need organize queue
		err = vpnapi.WgAdd(addr, data.WgPrivateRouterEnc, data.EndpointIPv4, data.IPv4CGNAT, data.IPv6ULA)
		if err != nil {
			return fmt.Errorf("wg add: %w", err)
		}
	}

	if bonly {
		return nil
	}

	for _, user := range data.Users {
		kd6 := netip.Addr{}
		if user.IsBrigadier {
			kd6 = data.KeydeskIPv6
		}

		// if we catch a slowdown problems we need organize queue
		err = vpnapi.WgPeerAdd(addr, user.WgPublicKey, data.WgPublicKey, user.WgPSKRouterEnc, user.IPv4Addr, user.IPv6Addr, kd6)
		if err != nil {
			return fmt.Errorf("wg add: %w", err)
		}
	}

	return nil
}
