package user

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/env"
	"github.com/vpngen/wordsgens/namesgenerator"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	WgRouterPSK      []byte
	WgShufflerPSK    []byte
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
		return fmt.Errorf("connect: %w", err)
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

		return fmt.Errorf("brigadier query: %w", err)
	}

	_, err = pgx.ForEachRow(rows, []any{&wg_public, &endpoint_ipv4, &dns_ipv4, &dns_ipv6, &ipv4_cgnat, &ipv6_ula}, func() error {
		//fmt.Printf("Brigade:\nwg_public: %v\nendpoint_ipv4: %v\ndns_ipv4: %v\ndns_ipv6: %v\nipv4_cgnat: %v\nipv6_ula: %v\n", wg_public, endpoint_ipv4, dns_ipv4, dns_ipv6, ipv4_cgnat, ipv6_ula)

		return nil
	})
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("brigadier row: %w", err)
	}

	u.EndpointWgPublic = wg_public
	u.EndpointIPv4 = endpoint_ipv4
	u.DNSv4 = dns_ipv4
	u.DNSv6 = dns_ipv6

	rows, err = tx.Query(context.Background(), fmt.Sprintf("SELECT user_id,user_callsign,user_ipv4,user_ipv6 FROM %s ORDER BY is_brigadier", (pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()))
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("user query: %w", err)
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
			return fmt.Errorf("convert: %w", err)
		}

		idL[id.String()] = struct{}{}
		ip4L[user_ipv4.String()] = struct{}{}
		ip6L[user_ipv6.String()] = struct{}{}

		return nil
	})
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("user row: %w", err)
	}

	if len(idL) >= MaxUsers {
		tx.Rollback(context.Background())

		return ErrUserLimit
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

	userNotify := &SrvNotify{
		T:        NotifyNewUser,
		Endpoint: NewEndpoint(u.EndpointIPv4),
		Brigade: SrvBrigade{
			ID:          env.Env.BrigadierID,
			IdentityKey: wg_public,
		},
		User: SrvUser{
			ID:          u.ID,
			WgPublicKey: u.WgPublicKey,
			IsBrigadier: u.Boss,
		},
	}

	notify, err := json.Marshal(userNotify)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("marshal notify: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		fmt.Sprintf(`INSERT INTO %s (user_id, user_callsign, is_brigadier, wg_public, wg_psk_router, wg_psk_shuffler, user_ipv4, user_ipv6) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			(pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()),
		u.ID, u.Name, u.Boss, `\x`+hex.EncodeToString(u.WgPublicKey), `\x`+hex.EncodeToString(u.WgRouterPSK), `\x`+hex.EncodeToString(u.WgShufflerPSK), u.IPv4.String(), u.IPv6.String(),
	)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		fmt.Sprintf(`INSERT INTO %s (user_id, person) VALUES ($1, $2 :: json);`,
			(pgx.Identifier{env.Env.BrigadierID, "persons"}).Sanitize()),
		u.ID, u.Person,
	)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("insert person: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		fmt.Sprintf(`INSERT INTO %s (user_id, limit_monthly_remaining, limit_monthly_reset_on, os_counter_mtime, os_counter_value) VALUES ($1, $2 :: int8 * 1024 * 1024 * 1024, 'now', 'now', 0);`,
			(pgx.Identifier{env.Env.BrigadierID, "quota"}).Sanitize()),
		u.ID, MonthlyQuotaRemainingGB,
	)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("insert quota: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		"SELECT pg_notify('vpngen', $1)",
		notify,
	)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("notify: %w", err)
	}

	tx.Commit(context.Background())

	return nil
}

func (us *userStorage) delete(id string) error {
	ctx := context.Background()

	tx, err := env.Env.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	rows, err := tx.Query(context.Background(), fmt.Sprintf("SELECT brigade.endpoint_ipv4, brigade.wg_public, users.wg_public FROM %s,%s WHERE users.user_id=$1", (pgx.Identifier{env.Env.BrigadierID, "brigade"}).Sanitize(), (pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()), id)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("user query: %w", err)
	}

	var (
		endpoint_ipv4     netip.Addr
		brigade_wg_public []byte
		user_wg_public    []byte
	)

	_, err = pgx.ForEachRow(rows, []any{&endpoint_ipv4, &brigade_wg_public, &user_wg_public}, func() error {

		return nil
	})
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("user row: %w", err)
	}

	userNotify := &SrvNotify{
		T:        NotifyDelUser,
		Endpoint: NewEndpoint(endpoint_ipv4),
		Brigade: SrvBrigade{
			ID:          env.Env.BrigadierID,
			IdentityKey: brigade_wg_public,
		},
		User: SrvUser{
			ID:          id,
			WgPublicKey: user_wg_public,
		},
	}

	notify, err := json.Marshal(userNotify)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("marshal notify: %w", err)
	}

	_, err = tx.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE user_id=$1", (pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()),
		id,
	)
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("delete users: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		"SELECT pg_notify('vpngen', $1)",
		notify,
	)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("notify: %w", err)
	}

	tx.Commit(ctx)

	return nil
}

func (us *userStorage) list() ([]*User, error) {
	ctx := context.Background()

	tx, err := env.Env.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	rows, err := tx.Query(ctx,
		fmt.Sprintf(
			`SELECT 
				users.user_id,
				users.user_callsign,
				users.is_brigadier,
				persons.person,
				quota.limit_monthly_remaining,
				quota.limit_monthly_reset_on,
				quota.last_activity,
				quota.last_origin,
				quota.last_asn,
				quota.p2p_slowdown_till
				FROM %s
				JOIN %s ON users.user_id = persons.user_id
				JOIN %s ON users.user_id = quota.user_id
			`,
			(pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize(),
			(pgx.Identifier{env.Env.BrigadierID, "persons"}).Sanitize(),
			(pgx.Identifier{env.Env.BrigadierID, "quota"}).Sanitize(),
		),
	)
	if err != nil {
		tx.Rollback(ctx)

		return nil, fmt.Errorf("users query: %w", err)
	}

	users := make([]*User, 0)

	var (
		user_id           string
		user_callsign     string
		is_brigadier      bool
		person            namesgenerator.Person
		lmr               pgtype.Int8
		lmro              pgtype.Date
		last_activity     pgtype.Timestamp
		last_origin       pgtype.Text
		last_asn          pgtype.Int4
		p2p_slowdown_till pgtype.Timestamp
	)

	_, err = pgx.ForEachRow(rows, []any{&user_id, &user_callsign, &is_brigadier, &person, &lmr, &lmro, &last_activity, &last_origin, &last_asn, &p2p_slowdown_till}, func() error {
		u := &User{}
		u.ID = user_id
		u.Name = user_callsign
		u.Boss = is_brigadier
		u.Person = person
		u.LastVisitHour = last_activity.Time
		u.MonthlyQuotaRemainingGB = float32(lmr.Int64 / 1024 / 1024 / 1024)
		u.ThrottlingTill = p2p_slowdown_till.Time
		u.LastVisitSubnet = last_origin.String
		//u.LastVisitASName = last_asn

		users = append(users, u)

		return nil
	})
	if err != nil {
		tx.Rollback(ctx)

		return nil, fmt.Errorf("users rows: %w", err)
	}

	tx.Commit(ctx)

	return users, nil
}
