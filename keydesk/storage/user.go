package storage

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/vpnapi"
	"github.com/vpngen/wordsgens/namesgenerator"
)

type APIUserResponse struct {
	Code                     string `json:"code"`
	OpenvpnClientCertificate string `json:"openvpn-client-certificate"`
}

// CreateUser - put user to the storage.
func (db *BrigadeStorage) CreateUser(
	fullname string,
	person namesgenerator.Person,
	isBrigadier,
	replaceBrigadier bool,
	wgPub,
	wgRouterPSK,
	wgShufflerPSK []byte,
	ovcCertRequestGzipBase64 string,
) (*UserConfig, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if isBrigadier && replaceBrigadier {
		fullname, person, err = db.removeBrigadier(data)
		if err != nil {
			return nil, fmt.Errorf("replace: %w", err)
		}
	}

	id, ipv4, ipv6, name, err := assembleUser(data, fullname, isBrigadier, db.MaxUsers)
	if err != nil {
		return nil, fmt.Errorf("assemble: %w", err)
	}

	ts := time.Now().UTC()

	caPem, err := kdlib.Unbase64Ungzip(data.OvCACertPemGzipBase64)
	if err != nil {
		return nil, fmt.Errorf("unbase64 ca: %w", err)
	}

	userconf := &UserConfig{
		ID:               id,
		Name:             name,
		IPv4:             ipv4,
		IPv6:             ipv6,
		EndpointWgPublic: data.WgPublicKey,
		EndpointIPv4:     data.EndpointIPv4,
		EndpointDomain:   data.EndpointDomain,
		EndPointPort:     data.EndpointPort,
		DNSv4:            data.DNSv4,
		DNSv6:            data.DNSv6,
		OvCACertPem:      string(caPem),
	}

	kd6 := netip.Addr{}
	if isBrigadier {
		kd6 = data.KeydeskIPv6
	}

	// if we catch a slowdown problems we need organize queue
	body, err := vpnapi.WgPeerAdd(db.actualAddrPort, db.calculatedAddrPort, wgPub, data.WgPublicKey, wgRouterPSK, userconf.IPv4, userconf.IPv6, kd6, ovcCertRequestGzipBase64)
	if err != nil {
		return nil, fmt.Errorf("wg add: %w", err)
	}

	payload := &APIUserResponse{}

	err = json.Unmarshal(body, payload)
	if err != nil {
		return nil, fmt.Errorf("resp body: %w", err)
	}

	userconf.OvClientCertPem = payload.OpenvpnClientCertificate

	data.Users = append(data.Users, &User{
		UserID:           userconf.ID,
		Name:             userconf.Name,
		CreatedAt:        ts, // creazy but can be data.KeydeskLastVisit
		IsBrigadier:      isBrigadier,
		IPv4Addr:         userconf.IPv4,
		IPv6Addr:         userconf.IPv6,
		WgPublicKey:      wgPub,
		WgPSKRouterEnc:   wgRouterPSK,
		WgPSKShufflerEnc: wgShufflerPSK,
		OvCSRGzipBase64:  ovcCertRequestGzipBase64,
		Person:           person,
		Quotas: Quota{
			CountersTotal: DateSummaryNetCounters{
				Ver: DateSummaryNetCountersVersion,
			},
			CountersWg: DateSummaryNetCounters{
				Ver: DateSummaryNetCountersVersion,
			},
			CountersIPSec: DateSummaryNetCounters{
				Ver: DateSummaryNetCountersVersion,
			},
			LimitMonthlyRemaining: uint64(db.MonthlyQuotaRemaining),
			LimitMonthlyResetOn:   kdlib.NextMonthlyResetOn(ts),
			Ver:                   QuotaVesrion,
		},
		Ver: UserVersion,
	})

	sort.Slice(data.Users, func(i, j int) bool {
		return data.Users[i].IsBrigadier || !data.Users[j].IsBrigadier && (data.Users[i].UserID.String() > data.Users[j].UserID.String())
	})

	if err := commitBrigade(f, data); err != nil {
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
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	wgPub := []byte{}
	for i, u := range data.Users {
		if u.UserID.String() == id && u.IsBrigadier == brigadier {
			wgPub = u.WgPublicKey
			data.Users = append(data.Users[:i], data.Users[i+1:]...)

			break
		}
	}

	// if we catch a slowdown problems we need organize queue
	err = vpnapi.WgPeerDel(db.actualAddrPort, db.calculatedAddrPort, wgPub, data.WgPublicKey)
	if err != nil {
		return fmt.Errorf("peer del: %w", err)
	}

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

func (db *BrigadeStorage) removeBrigadier(data *Brigade) (string, namesgenerator.Person, error) {
	var (
		fullname string
		person   namesgenerator.Person
	)

	for i, user := range data.Users {
		if user.IsBrigadier {
			fullname, person = strings.TrimLeft(user.Name, "0123456789 "), user.Person

			wgPub := user.WgPublicKey
			data.Users = append(data.Users[:i], data.Users[i+1:]...)

			// if we catch a slowdown problems we need organize queue
			err := vpnapi.WgPeerDel(db.actualAddrPort, db.calculatedAddrPort, wgPub, data.WgPublicKey)
			if err != nil {
				return "", namesgenerator.Person{}, fmt.Errorf("peer del: %w", err)
			}

			break
		}
	}

	return fullname, person, nil
}

// ListUsers - list users.
func (db *BrigadeStorage) ListUsers() ([]*User, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.KeydeskFirstVisit.IsZero() {
		data.KeydeskFirstVisit = time.Now().UTC()

		if err := commitBrigade(f, data); err != nil {
			return nil, fmt.Errorf("save: %w", err)
		}
	}

	return data.Users, nil
}

func (db *BrigadeStorage) GetUsersStats() (StatsCountersStack, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return StatsCountersStack{}, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	return data.StatsCountersStack, nil
}
