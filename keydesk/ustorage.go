package keydesk

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
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
)

// BrigadeStorage - brigade file storage.
type BrigadeStorage struct {
	BrigadeFilename string
	StatsFilename   string
}

func (db *BrigadeStorage) userPut(fullname string, person namesgenerator.Person, IsBrigadier bool, wgPub, wgRouterPSK, wgShufflerPSK []byte) (*UserConfig, error) {
	var data *Brigade

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	err = f.Decoder().Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
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
		Quota:            Quota{LimitMonthlyRemaining: MonthlyQuotaRemainingGB},
	})

	sort.Slice(data.Users, func(i, j int) bool {
		return data.Users[i].IsBrigadier || data.Users[i].UserID.String() > data.Users[j].UserID.String()
	})

	f.Encoder().SetIndent(" ", " ")

	err = f.Encoder().Encode(data)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	// !!! DO API CALL
	// if we catch a slowdown problems we need organize queue

	f.Commit()

	return userconf, nil
}

func (db *BrigadeStorage) userRemove(id string, brigadier bool) error {
	var data *Brigade

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	err = f.Decoder().Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	for i, u := range data.Users {
		if u.UserID.String() == id && u.IsBrigadier == brigadier {
			data.Users = append(data.Users, data.Users[i+1:]...)

			break
		}
	}

	f.Encoder().SetIndent(" ", " ")

	err = f.Encoder().Encode(data)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// !!! DO API CALL
	// if we catch a slowdown problems we need organize queue

	f.Commit()

	return nil
}

func (db *BrigadeStorage) userList() ([]*User, error) {
	var data *Brigade

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	err = f.Decoder().Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return data.Users, nil
}
