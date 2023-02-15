package keydesk

import (
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/epapi"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// Filenames.
const (
	BrigadeFilename = "brigade.json"
	StatsFilename   = "stats.json"
)

var (
	// ErrUserLimit - maximun user num exeeded.
	ErrUserLimit = errors.New("num user limit exeeded")
	// ErrUserCollision - user name collision.
	ErrUserCollision = errors.New("username exists")
	// ErrUnknownBrigade - brigade ID mismatch.
	ErrUnknownBrigade = errors.New("unknown brigade")
)

// BrigadeStorage - brigade file storage.
type BrigadeStorage struct {
	BrigadeID       string
	BrigadeFilename string
	StatsFilename   string
	APIAddrPort     netip.AddrPort
}

// BrigadePut - create brigade config.
func (db *BrigadeStorage) BrigadePut(config *BrigadeConfig, wgPub, wgRouterPriv, wgShufflerPriv []byte) error {
	if config.BrigadeID != db.BrigadeID {
		return fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	addr := db.APIAddrPort
	if addr.Addr().IsValid() && addr.Addr().IsUnspecified() {
		addr = epapi.CalcAPIAddrPort(config.EndpointIPv4)
	}

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	data := Brigade{
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

	err = f.Encoder(" ", " ").Encode(data)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// if we catch a slowdown problems we need organize queue
	err = epapi.WgAdd(addr, wgRouterPriv, config.EndpointIPv4, config.IPv4CGNAT, config.IPv6ULA)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	f.Commit()

	return nil
}

// BrigadeRemove - remove brigade.
func (db *BrigadeStorage) BrigadeRemove() error {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	// !!! DO API CALL
	// if we catch a slowdown problems we need organize queue

	f.Commit()

	return nil
}

// UserPut - put user to the storage.
func (db *BrigadeStorage) UserPut(fullname string, person namesgenerator.Person, IsBrigadier bool, wgPub, wgRouterPSK, wgShufflerPSK []byte) (*UserConfig, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	userconf := &UserConfig{
		EndpointWgPublic: data.WgPublicKey,
		EndpointIPv4:     data.EndpointIPv4,
		DNSv4:            data.DNSv4,
		DNSv6:            data.DNSv6,
	}

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
			return nil, ErrUserCollision
		}

		idL[user.UserID.String()] = struct{}{}
		ip4L[user.IPv4Addr.String()] = struct{}{}
		ip6L[user.IPv6Addr.String()] = struct{}{}

	}

	if len(idL) >= MaxUsers {
		return nil, ErrUserLimit
	}

	for {
		id := uuid.New()

		if _, ok := idL[id.String()]; !ok {
			userconf.ID = id

			break
		}
	}

	for {
		ip := kdlib.RandomAddrIPv4(data.IPv4CGNAT)
		if kdlib.IsZeroEnding(ip) {
			continue
		}

		if _, ok := ip4L[ip.String()]; !ok {
			userconf.IPv4 = ip

			break
		}
	}

	for {
		ip := kdlib.RandomAddrIPv6(data.IPv6ULA)
		if kdlib.IsZeroEnding(ip) {
			continue
		}

		if _, ok := ip6L[ip.String()]; !ok {
			userconf.IPv6 = ip

			break
		}
	}

	userNum := blurIpv4Addr(userconf.IPv4, data.IPv4CGNAT.Bits(), extractUint32Salt(data.BrigadeID))
	userconf.Name = fmt.Sprintf("%03d %s", userNum, fullname)

	data.Users = append(data.Users, &User{
		UserID:           userconf.ID,
		Name:             userconf.Name,
		CreatedAt:        time.Now(),
		IsBrigadier:      IsBrigadier,
		IPv4Addr:         userconf.IPv4,
		IPv6Addr:         userconf.IPv6,
		WgPublicKey:      wgPub,
		WgPSKRouterEnc:   wgRouterPSK,
		WgPSKShufflerEnc: wgShufflerPSK,
		Person:           person,
		Quota:            Quota{LimitMonthlyRemaining: MonthlyQuotaRemaining},
	})

	sort.Slice(data.Users, func(i, j int) bool {
		return data.Users[i].IsBrigadier || data.Users[i].UserID.String() > data.Users[j].UserID.String()
	})

	err = f.Encoder(" ", " ").Encode(data)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	// !!! DO API CALL
	// if we catch a slowdown problems we need organize queue

	f.Commit()

	return userconf, nil
}

// UserRemove - remove user from the storage.
func (db *BrigadeStorage) UserRemove(id string, brigadier bool) error {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	for i, u := range data.Users {
		if u.UserID.String() == id && u.IsBrigadier == brigadier {
			data.Users = append(data.Users[:i], data.Users[i+1:]...)

			break
		}
	}

	err = f.Encoder(" ", " ").Encode(data)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// !!! DO API CALL
	// if we catch a slowdown problems we need organize queue

	f.Commit()

	return nil
}

// UserList - list users.
func (db *BrigadeStorage) UserList() ([]*User, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return data.Users, nil
}
