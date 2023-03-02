package storage

import (
	"fmt"
	"net/netip"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/vapnapi"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// CreateUser - put user to the storage.
func (db *BrigadeStorage) CreateUser(
	fullname string,
	person namesgenerator.Person,
	isBrigadier,
	replaceBrigadier bool,
	wgPub,
	wgRouterPSK,
	wgShufflerPSK []byte,
) (*UserConfig, error) {
	dt, data, stat, addr, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	if isBrigadier && replaceBrigadier {
		err := db.removeBrigadier(data, addr)
		if err != nil {
			return nil, fmt.Errorf("replace: %w", err)
		}
	}

	id, ipv4, ipv6, name, err := assembleUser(data, fullname, isBrigadier, db.MaxUsers)
	if err != nil {
		return nil, fmt.Errorf("assemble: %w", err)
	}

	userconf := &UserConfig{
		ID:               id,
		Name:             name,
		IPv4:             ipv4,
		IPv6:             ipv6,
		EndpointWgPublic: data.WgPublicKey,
		EndpointIPv4:     data.EndpointIPv4,
		DNSv4:            data.DNSv4,
		DNSv6:            data.DNSv6,
	}

	data.Users = append(data.Users, &User{
		UserID:           userconf.ID,
		Name:             userconf.Name,
		CreatedAt:        time.Now(), // creazy but can be data.KeydeskLastVisit
		IsBrigadier:      isBrigadier,
		IPv4Addr:         userconf.IPv4,
		IPv6Addr:         userconf.IPv6,
		WgPublicKey:      wgPub,
		WgPSKRouterEnc:   wgRouterPSK,
		WgPSKShufflerEnc: wgShufflerPSK,
		Person:           person,
		Quota: Quota{
			Counters: NetCounters{
				Ver: NetCountersVersion,
			},
			LimitMonthlyRemaining: uint64(db.MonthlyQuotaRemaining),
			Ver:                   QuotaVesrion,
		},
		Ver: UserVersion,
	})

	sort.Slice(data.Users, func(i, j int) bool {
		return data.Users[i].IsBrigadier || !data.Users[j].IsBrigadier && (data.Users[i].UserID.String() > data.Users[j].UserID.String())
	})

	kd6 := netip.Addr{}
	if isBrigadier {
		kd6 = data.KeydeskIPv6
	}

	// if we catch a slowdown problems we need organize queue
	err = vapnapi.WgPeerAdd(addr, wgPub, data.WgPublicKey, wgRouterPSK, userconf.IPv4, userconf.IPv6, kd6)
	if err != nil {
		return nil, fmt.Errorf("wg add: %w", err)
	}

	aggrStat(data, stat, db.ActivityPeriod)

	dt.save(data, stat)
	if err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	return userconf, nil
}

func assembleUser(data *Brigade, fullname string, isBrigadier bool, maxUsers int) (uuid.UUID, netip.Addr, netip.Addr, string, error) {
	var (
		ipv4, ipv6 netip.Addr
		uid        uuid.UUID
	)

	idL := make(map[string]struct{})

	// put self and broadcast addresses.
	ip4L := map[string]struct{}{
		data.IPv4CGNAT.Addr().String():                {},
		kdlib.LastPrefixIPv4(data.IPv4CGNAT).String(): {},
	}

	ip6L := map[string]struct{}{
		data.IPv6ULA.Addr().String():                {},
		kdlib.LastPrefixIPv6(data.IPv6ULA).String(): {},
	}

	for _, user := range data.Users {
		if user.Name == fullname {
			return uid, ipv4, ipv6, "", ErrUserCollision
		}

		if isBrigadier && user.IsBrigadier {
			return uid, ipv4, ipv6, "", ErrBrigadierCollision
		}

		idL[user.UserID.String()] = struct{}{}
		ip4L[user.IPv4Addr.String()] = struct{}{}
		ip6L[user.IPv6Addr.String()] = struct{}{}
	}

	if len(idL) >= maxUsers {
		return uid, ipv4, ipv6, "", ErrUserLimit
	}

	for {
		id := uuid.New()
		if _, ok := idL[id.String()]; !ok {
			uid = id

			break
		}
	}

	for {
		ip := kdlib.RandomAddrIPv4(data.IPv4CGNAT)
		if kdlib.IsZeroEnding(ip) {
			continue
		}

		if _, ok := ip4L[ip.String()]; !ok {
			ipv4 = ip

			break
		}
	}

	for {
		ip := kdlib.RandomAddrIPv6(data.IPv6ULA)
		if kdlib.IsZeroEnding(ip) {
			continue
		}

		if _, ok := ip6L[ip.String()]; !ok {
			ipv6 = ip

			break
		}
	}

	name := fmt.Sprintf("%03d %s",
		kdlib.BlurIpv4Addr(ipv4, data.IPv4CGNAT.Bits(), kdlib.ExtractUint32Salt(data.BrigadeID)),
		fullname)

	return uid, ipv4, ipv6, name, nil
}

// DeleteUser - remove user from the storage.
func (db *BrigadeStorage) DeleteUser(id string, brigadier bool) error {
	dt, data, stat, addr, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	wgPub := []byte{}
	for i, u := range data.Users {
		if u.UserID.String() == id && u.IsBrigadier == brigadier {
			wgPub = u.WgPublicKey
			data.Users = append(data.Users[:i], data.Users[i+1:]...)

			break
		}
	}

	// if we catch a slowdown problems we need organize queue
	err = vapnapi.WgPeerDel(addr, wgPub, data.WgPublicKey)
	if err != nil {
		return fmt.Errorf("peer del: %w", err)
	}

	aggrStat(data, stat, db.ActivityPeriod)

	dt.save(data, stat)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

func (db *BrigadeStorage) removeBrigadier(data *Brigade, addr netip.AddrPort) error {
	for i, user := range data.Users {
		if user.IsBrigadier {
			wgPub := user.WgPublicKey
			data.Users = append(data.Users[:i], data.Users[i+1:]...)

			// if we catch a slowdown problems we need organize queue
			err := vapnapi.WgPeerDel(addr, wgPub, data.WgPublicKey)
			if err != nil {
				return fmt.Errorf("peer del: %w", err)
			}

			break
		}
	}

	return nil
}

// ListUsers - list users.
func (db *BrigadeStorage) ListUsers() ([]*User, error) {
	dt, data, stat, _, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer dt.close()

	ts := time.Now()
	data.KeydeskLastVisit = ts
	stat.KeydeskLastVisit = ts

	dt.save(data, stat)
	if err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	return data.Users, nil
}
