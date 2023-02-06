package keydesk

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// MonthlyQuotaRemainingGB - .
const MonthlyQuotaRemainingGB = 100

var (
	// ErrUserLimit - maximun user num exeeded.
	ErrUserLimit = errors.New("num user limit exeeded")
	// ErrUserCollision - user name collision.
	ErrUserCollision = errors.New("username exists")
)

type userStorage struct {
	sync.Mutex
	m  map[string]*User
	nm map[string]struct{}
}

var storage = &userStorage{
	m:  make(map[string]*User),
	nm: make(map[string]struct{}),
}

func (us *userStorage) put(fullname string, person namesgenerator.Person, IsBrigadier bool, wgPub, wgRouterPSK, wgShufflerPSK []byte) (*UserConfig, error) {
	data := &Brigade{
		Users: []User{},
	} // !!!

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

	return userconf, nil
}

func (us *userStorage) delete(id string, boss bool) error {
	/// !!!
	return nil
}

func (us *userStorage) list() ([]*User, error) {
	// !!!
	return []*User{}, nil
}
