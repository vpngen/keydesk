package keydesk

import (
	"errors"
	"fmt"
	"io"
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
	// ErrBrigadierCollision - try to add more than one.
	ErrBrigadierCollision = errors.New("brigadier already exists")
	// ErrUnknownBrigade - brigade ID mismatch.
	ErrUnknownBrigade = errors.New("unknown brigade")
	// ErrBrigadeAlreadyExists - brigade file exists unexpectabily.
	ErrBrigadeAlreadyExists = errors.New("already exists")
)

// BrigadeStorage - brigade file storage.
type BrigadeStorage struct {
	BrigadeID       string
	BrigadeFilename string
	StatsFilename   string
	APIAddrPort     netip.AddrPort
}

func (db *BrigadeStorage) openWithReading() (*kdlib.FileDb, *Brigade, netip.AddrPort, error) {
	addr := netip.AddrPort{}

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, nil, addr, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		f.Close()

		return nil, nil, addr, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, nil, addr, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	addr = db.APIAddrPort
	if addr.Addr().IsValid() && addr.Addr().IsUnspecified() {
		addr = epapi.CalcAPIAddrPort(data.EndpointIPv4)
	}

	return f, data, addr, nil
}

func (db *BrigadeStorage) openWithoutReading(brigadeID string) (*kdlib.FileDb, *Brigade, error) {
	if brigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	switch err {
	case nil:
		f.Close()

		return nil, nil, fmt.Errorf("integrity: %w", ErrBrigadeAlreadyExists)
	case io.EOF:
		break
	default:
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	return f, data, nil
}

func (db *BrigadeStorage) save(f *kdlib.FileDb, data *Brigade) error {
	err := f.Encoder(" ", " ").Encode(data)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	err = f.Commit()
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

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

// CreateUser - put user to the storage.
func (db *BrigadeStorage) CreateUser(fullname string, person namesgenerator.Person, isBrigadier, rewriteBrigadier bool, wgPub, wgRouterPSK, wgShufflerPSK []byte) (*UserConfig, error) {
	f, data, addr, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if isBrigadier && rewriteBrigadier {
		err := db.removeBrigadier(data, addr)
		if err != nil {
			return nil, fmt.Errorf("replace: %w", err)
		}
	}

	id, ipv4, ipv6, name, err := assembleUser(data, fullname, isBrigadier)
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
		CreatedAt:        time.Now(),
		IsBrigadier:      isBrigadier,
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

	kd6 := netip.Addr{}
	if isBrigadier {
		kd6 = data.KeydeskIPv6
	}

	// if we catch a slowdown problems we need organize queue
	err = epapi.PeerAdd(addr, wgPub, data.WgPublicKey, wgRouterPSK, userconf.IPv4, userconf.IPv6, kd6)
	if err != nil {
		return nil, fmt.Errorf("wg add: %w", err)
	}

	db.save(f, data)
	if err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	return userconf, nil
}

func assembleUser(data *Brigade, fullname string, isBrigadier bool) (uuid.UUID, netip.Addr, netip.Addr, string, error) {
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

	if len(idL) >= MaxUsers {
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
		blurIpv4Addr(ipv4, data.IPv4CGNAT.Bits(), extractUint32Salt(data.BrigadeID)),
		fullname)

	return uid, ipv4, ipv6, name, nil
}

// DeleteUser - remove user from the storage.
func (db *BrigadeStorage) DeleteUser(id string, brigadier bool) error {
	f, data, addr, err := db.openWithReading()
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
	err = epapi.PeerDel(addr, wgPub, data.WgPublicKey)
	if err != nil {
		return fmt.Errorf("peer del: %w", err)
	}

	db.save(f, data)
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
			err := epapi.PeerDel(addr, wgPub, data.WgPublicKey)
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
	f, data, _, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	return data.Users, nil
}
