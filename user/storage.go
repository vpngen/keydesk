package user

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keykeeper/env"
	"github.com/vpngen/wordsgens/namesgenerator"

	"github.com/jackc/pgx/v5"
)

// MonthlyQuotaRemainingGB - .
const MonthlyQuotaRemainingGB = 100

var (
	// ErrUserLimit - maximun user num exeeded.
	ErrUserLimit = errors.New("num user limit exeeded")
	// ErrUserCollision - user name collision.
	ErrUserCollision = errors.New("username exists")
)

// User - user structure.
type User struct {
	ID                      string
	Name                    string
	Person                  namesgenerator.Person
	MonthlyQuotaRemainingGB float32
	Problems                []string
	ThrottlingTill          time.Time
	LastVisitHour           time.Time
	LastVisitSubnet         string
	LastVisitASName         string
	LastVisitASCountry      string
	Boss                    bool
}

// UserConfig - new user structure.
type UserConfig struct {
	ID               string
	Name             string
	Person           namesgenerator.Person
	Boss             bool
	WgPublicKey      []byte
	WgRouterPriv     []byte
	WgShufflerPriv   []byte
	DNSv4, DNSv6     netip.Addr
	IPv4, IPv6       netip.Addr
	EndpointIPv4     netip.Addr
	EndpointWgPublic []byte
}

type userStorage struct {
	sync.Mutex
	m  map[string]*User
	nm map[string]struct{}
}

var storage = &userStorage{
	m:  make(map[string]*User),
	nm: make(map[string]struct{}),
}

func (us *userStorage) put(u *UserConfig) error {
	tx, err := env.Env.DB.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("Can't connect: %w", err)
	}

	var (
		wg_public          []byte
		endpoint_ipv4      netip.Addr
		dns_ipv4, dns_ipv6 netip.Addr
		ipv4_cgnat         netip.Prefix
		ipv6_ula           netip.Prefix
	)

	rows, err := tx.Query(context.Background(), fmt.Sprintf("SELECT wg_public,endpoint_ipv4,dns_ipv4,dns_ipv6,ipv4_cgnat,ipv6_ula FROM %s FOR UPDATE", (pgx.Identifier{env.Env.BrigadierID, "brigade"}).Sanitize()))
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("Can't brigadier query: %w", err)
	}

	_, err = pgx.ForEachRow(rows, []any{&wg_public, &endpoint_ipv4, &dns_ipv4, &dns_ipv6, &ipv4_cgnat, &ipv6_ula}, func() error {
		//fmt.Printf("Brigade:\nwg_public: %v\nendpoint_ipv4: %v\ndns_ipv4: %v\ndns_ipv6: %v\nipv4_cgnat: %v\nipv6_ula: %v\n", wg_public, endpoint_ipv4, dns_ipv4, dns_ipv6, ipv4_cgnat, ipv6_ula)

		return nil
	})
	if err != nil {
		tx.Rollback(context.Background())

		return err
	}

	u.EndpointWgPublic = wg_public
	u.EndpointIPv4 = endpoint_ipv4
	u.DNSv4 = dns_ipv4
	u.DNSv6 = dns_ipv6

	rows, err = tx.Query(context.Background(), fmt.Sprintf("SELECT user_id,user_callsign,user_ipv4,user_ipv6 FROM %s ORDER BY is_brigadier", (pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()))
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("Can't users query: %w", err)
	}

	var (
		user_id              []byte
		user_callsign        string
		user_ipv4, user_ipv6 netip.Addr
	)

	idL := make(map[string]struct{})
	ip4L := make(map[string]struct{})
	ip6L := make(map[string]struct{})

	_, err = pgx.ForEachRow(rows, []any{&user_id, &user_callsign, &user_ipv4, &user_ipv6}, func() error {
		if user_callsign == u.Name {
			return ErrUserCollision
		}

		id, err := uuid.FromBytes(user_id)
		if err != nil {
			return fmt.Errorf("cant convert: %w", err)
		}

		idL[id.String()] = struct{}{}
		ip4L[user_ipv4.String()] = struct{}{}
		ip6L[user_ipv6.String()] = struct{}{}

		return nil
	})
	if err != nil {
		tx.Rollback(context.Background())

		return err
	}

	for {
		u.ID = uuid.New().String()

		if _, ok := idL[u.ID]; !ok {
			break
		}
	}

	for {
		u.IPv4 = RandomAddrIPv4(ipv4_cgnat)

		if _, ok := ip4L[u.IPv4.String()]; !ok {
			break
		}
	}

	for {
		u.IPv6 = RandomAddrIPv6(ipv6_ula)

		if _, ok := ip6L[u.IPv6.String()]; !ok {
			break
		}
	}

	_, err = tx.Exec(context.Background(),
		fmt.Sprintf(`INSERT INTO %s (user_id, user_callsign, is_brigadier, wg_public, wg_psk_router, wg_psk_shuffler, user_ipv4, user_ipv6) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			(pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()),
		u.ID, u.Name, u.Boss, `\x`+hex.EncodeToString(u.WgPublicKey), `\x`+hex.EncodeToString(u.WgRouterPriv), `\x`+hex.EncodeToString(u.WgShufflerPriv), u.IPv4.String(), u.IPv6.String(),
	)
	if err != nil {
		tx.Rollback(context.Background())

		return err
	}

	_, err = tx.Exec(context.Background(),
		fmt.Sprintf(`INSERT INTO %s (user_id, limit_monthly_remaining, limit_monthly_reset_on, os_counter_mtime, os_counter_value) VALUES ($1, $2 :: int8 * 1024 * 1024 * 1024, 'now', 'now', 0);`,
			(pgx.Identifier{env.Env.BrigadierID, "quota"}).Sanitize()),
		u.ID, MonthlyQuotaRemainingGB,
	)
	if err != nil {
		tx.Rollback(context.Background())

		return err
	}

	_, err = tx.Exec(context.Background(),
		fmt.Sprintf("SELECT pg_notify('vpngen', '{\"t\":\"new-user\",\"brigade_id\":%q,\"user_id\":%q}')",
			env.Env.BrigadierID, u.ID,
		),
	)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("notify: %w", err)
	}

	tx.Commit(context.Background())

	return nil
}

func (us *userStorage) delete(id string) bool {
	us.Lock()
	defer us.Unlock()

	if u, ok := us.m[id]; ok {
		if u.Boss {
			return false
		}

		if _, ok := us.nm[u.Name]; ok {
			delete(us.nm, u.Name)
		}

		delete(us.m, id)
	}

	return true
}

func (us *userStorage) list() []*User {
	us.Lock()
	defer us.Unlock()

	res := make([]*User, 0, len(us.m))

	for _, v := range us.m {
		res = append(res, v)
	}

	return res
}
