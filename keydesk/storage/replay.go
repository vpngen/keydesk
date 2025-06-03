package storage

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/vpngen/keydesk/vpnapi"
)

// ReplayBrigade - create brigade config.
func (db *BrigadeStorage) ReplayBrigade(fresh, bonly, uonly, delayed, donly bool) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if !donly {
		if fresh {
			// if we catch a slowdown problems we need organize queue
			err = vpnapi.WgDel(data.BrigadeID, db.actualAddrPort, db.calculatedAddrPort, data.WgPrivateRouterEnc)
			if err != nil {
				return fmt.Errorf("wg del: %w", err)
			}
		}

		proto0Decoy := []string{}
		if data.Proto0FakeDomain != "" {
			proto0Decoy = append(proto0Decoy, data.Proto0FakeDomain)
		}

		if len(data.Proto0FakeDomains) > 0 {
			proto0Decoy = append(proto0Decoy, data.Proto0FakeDomains...)
		}

		if !uonly {
			// if we catch a slowdown problems we need organize queue
			err = vpnapi.WgAdd(
				data.BrigadeID,
				db.actualAddrPort,
				db.calculatedAddrPort,
				data.WgPrivateRouterEnc,
				data.EndpointIPv4,
				data.EndpointPort,
				data.IPv4CGNAT,
				data.IPv6ULA,
				data.CloakFakeDomain,
				data.OvCACertPemGzipBase64,
				data.OvCAKeyRouterEnc,
				data.IPSecPSKRouterEnc,
				data.OutlinePort,
				strings.Join(proto0Decoy, ","),
			)
			if err != nil {
				return fmt.Errorf("wg add: %w", err)
			}
		}

		if bonly {
			return nil
		}
	}

	if delayed || donly {
	OUTER:
		for {
			for i, user := range data.Users {
				if user.DelayedDeletion {
					if err = vpnapi.WgPeerDel(
						data.BrigadeID,
						db.actualAddrPort, db.calculatedAddrPort,
						user.WgPublicKey, data.WgPublicKey,
					); err != nil {
						return fmt.Errorf("wg del: %w", err)
					}

					data.Users = append(data.Users[:i], data.Users[i+1:]...)

					continue OUTER
				}
			}

			break
		}
	}

	for _, user := range data.Users {
		kd6 := netip.Addr{}
		if user.IsBrigadier {
			kd6 = data.KeydeskIPv6
		}

		if !donly && !delayed && (user.DelayedCreation || user.DelayedDeletion || user.DelayedReplay || user.DelayedBlocking) {
			continue
		}

		if donly || delayed {
			switch {
			case user.DelayedBlocking:
				user.DelayedBlocking = false

				if err = vpnapi.WgPeerDel(
					data.BrigadeID,
					db.actualAddrPort, db.calculatedAddrPort,
					user.WgPublicKey, data.WgPublicKey,
				); err != nil {
					return fmt.Errorf("wg del: %w", err)
				}
			case user.DelayedReplay:
				user.DelayedReplay = false
				user.DelayedCreation = true

				if err = vpnapi.WgPeerDel(
					data.BrigadeID,
					db.actualAddrPort, db.calculatedAddrPort,
					user.WgPublicKey, data.WgPublicKey,
				); err != nil {
					return fmt.Errorf("wg del: %w", err)
				}
			}
		}

		if user.IsBlocked {
			continue
		}

		if donly && (!user.DelayedCreation && !user.DelayedReplay && !user.DelayedBlocking) {
			continue
		}

		// if we catch a slowdown problems we need organize queue
		if _, err = vpnapi.WgPeerAdd(
			data.BrigadeID,
			db.actualAddrPort, db.calculatedAddrPort,
			user.WgPublicKey, data.WgPublicKey, user.WgPSKRouterEnc,
			user.IPv4Addr, user.IPv6Addr, kd6,
			user.OvCSRGzipBase64, user.CloakByPassUIDRouterEnc,
			user.IPSecUsernameRouterEnc, user.IPSecPasswordRouterEnc,
			user.OutlineSecretRouterEnc, user.Proto0SecretRouterEnc,
		); err != nil {
			return fmt.Errorf("wg add: %w", err)
		}

		if (donly || delayed) && user.DelayedCreation {
			user.DelayedCreation = false
		}
	}

	// beacuse we need to save changes only if delayed || donly flags are set
	if donly || delayed {
		if err := commitBrigade(f, data); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	return nil
}
