package storage

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/netip"
	"os"
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

const testCert = `-----BEGIN CERTIFICATE-----
MIIChjCCAeigAwIBAgIUHYRJHPNW+eqW3TkSaWhpRxqyk68wCgYIKoZIzj0EAwIw
VDELMAkGA1UEBhMCUlUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGElu
dGVybmV0IFdpZGdpdHMgUHR5IEx0ZDENMAsGA1UEAwwEVGVzdDAgFw0yMzA4MTcx
NDE0MTRaGA8yMDUxMDEwMjE0MTQxNFowVDELMAkGA1UEBhMCUlUxEzARBgNVBAgM
ClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEN
MAsGA1UEAwwEVGVzdDCBmzAQBgcqhkjOPQIBBgUrgQQAIwOBhgAEADrZB/oUNXuU
kAoyC1DCoqWnp0pdJx5GuxqxAJD9uMYOS05G3PjAboesJohnoFGOld2Zh2Kuj6OJ
ULh9hTj14eB7AZT4YX/vjA/odBS/Bu9PSjMiyrwTCms1hkMl2EvS06Hc3ElrjsuY
YMma/Chd8G+GAX12ijNO7BMlhLjhoZm383oao1MwUTAdBgNVHQ4EFgQU3x7cM6Kd
TEJN6KQvc0cHjAODOCwwHwYDVR0jBBgwFoAU3x7cM6KdTEJN6KQvc0cHjAODOCww
DwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgOBiwAwgYcCQUtlwuBJgT4gSGfH
yax9nYcFz6DzTaXWe3CZG0oLReUTrP88CeYfevWAvO7etL8IRKr48OWWm+sARDzY
GH/IDRigAkIBI45wN1CUGzzBjF8/faxNy6XWhcSkFZW7oCRR0MWaL6bn69naej8K
0msNdKBh0Uyk4SK0q+4NlBMTgoimpXcNdk8=
-----END CERTIFICATE-----`

// CreateUser - put user to the storage.
func (db *BrigadeStorage) CreateUser(
	uid uuid.UUID,
	vpnCfgs *ConfigsImplemented,
	fullname string,
	person namesgenerator.Person,
	isBrigadier,
	replaceBrigadier bool,
	wgPub,
	wgRouterPSK,
	wgShufflerPSK []byte,
	ovcCertRequestGzipBase64 string,
	cloakBypassUIDRouterEnc string,
	cloakBypassUIDShufflerEnc string,
	ipsecUsernameRouterEnc string,
	ipsecPasswordRouterEnc string,
	ipsecUsernameShufflerEnc string,
	ipsecPasswordShufflerEnc string,
	outlineSecretRouterEnc string,
	outlineSecretShufflerEnc string,
	proto0SecretRouterEnc string,
	proto0SecreShufflerEnc string,
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

	id, ipv4, ipv6, name, err := assembleUser(uid, data, fullname, isBrigadier, db.MaxUsers)
	if err != nil {
		return nil, fmt.Errorf("assemble: %w", err)
	}

	ts := time.Now().UTC()

	userconf := &UserConfig{
		ID:               id,
		Name:             name,
		IPv4:             ipv4,
		IPv6:             ipv6,
		EndpointWgPublic: data.WgPublicKey,
		EndpointIPv4:     data.EndpointIPv4,
		EndpointDomain:   data.EndpointDomain,
		EndpointPort:     data.EndpointPort,
		DNSv4:            data.DNSv4,
		DNSv6:            data.DNSv6,
		IPSecPSK:         data.IPSecPSK,
	}

	kd6 := netip.Addr{}
	if isBrigadier {
		kd6 = data.KeydeskIPv6
	}

	switch len(vpnCfgs.Ovc) {
	case 0:
		ovcCertRequestGzipBase64 = ""
	default:
		caPem, err := kdlib.Unbase64Ungzip(data.OvCACertPemGzipBase64)
		if err != nil {
			return nil, fmt.Errorf("unbase64 ca: %w", err)
		}

		userconf.OvCACertPem = string(caPem)
	}

	if len(vpnCfgs.Outline) > 0 {
		userconf.OutlinePort = data.OutlinePort
	}

	if len(vpnCfgs.Outline) > 0 || len(vpnCfgs.Ovc) > 0 {
		userconf.CloakFakeDomain = data.CloakFakeDomain
	}

	if len(vpnCfgs.Proto0) > 0 {
		switch len(data.Proto0FakeDomains) {
		case 0:
			userconf.Proto0FakeDomain = data.Proto0FakeDomain
		default:
			x, err := rand.Int(rand.Reader, big.NewInt(int64(len(data.Proto0FakeDomains))))
			if err != nil {
				panic(err)
			}

			userconf.Proto0FakeDomain = data.Proto0FakeDomains[x.Int64()]
		}

		userconf.Proto0Port = data.Proto0Port
	}

	// if we catch a slowdown problems we need organize queue
	body, err := vpnapi.WgPeerAdd(
		data.BrigadeID,
		db.actualAddrPort, db.calculatedAddrPort,
		wgPub, data.WgPublicKey, wgRouterPSK,
		userconf.IPv4, userconf.IPv6, kd6,
		ovcCertRequestGzipBase64, cloakBypassUIDRouterEnc,
		ipsecUsernameRouterEnc, ipsecPasswordRouterEnc,
		outlineSecretRouterEnc,
		proto0SecretRouterEnc,
	)
	if err != nil {
		return nil, fmt.Errorf("wg peer add: %w", err)
	}

	payload := &APIUserResponse{}

	switch db.actualAddrPort.Addr().IsValid() {
	case true:
		if err := json.Unmarshal(body, payload); err != nil {
			return nil, fmt.Errorf("resp body: %w", err)
		}
	default:
		payload.Code = "0"
		payload.OpenvpnClientCertificate = testCert
	}

	userconf.OvClientCertPem = payload.OpenvpnClientCertificate

	data.Users = append(data.Users, &User{
		UserID:                    userconf.ID,
		Name:                      userconf.Name,
		CreatedAt:                 ts, // creazy but can be data.KeydeskLastVisit
		IsBrigadier:               isBrigadier,
		IsSocket:                  false,
		IPv4Addr:                  userconf.IPv4,
		IPv6Addr:                  userconf.IPv6,
		WgPublicKey:               wgPub,
		WgPSKRouterEnc:            wgRouterPSK,
		WgPSKShufflerEnc:          wgShufflerPSK,
		OvCSRGzipBase64:           ovcCertRequestGzipBase64,
		CloakByPassUIDRouterEnc:   cloakBypassUIDRouterEnc,
		CloakByPassUIDShufflerEnc: cloakBypassUIDShufflerEnc,
		IPSecUsernameRouterEnc:    ipsecUsernameRouterEnc,
		IPSecUsernameShufflerEnc:  ipsecUsernameShufflerEnc,
		IPSecPasswordRouterEnc:    ipsecPasswordRouterEnc,
		IPSecPasswordShufflerEnc:  ipsecPasswordShufflerEnc,
		OutlineSecretRouterEnc:    outlineSecretRouterEnc,
		OutlineSecretShufflerEnc:  outlineSecretShufflerEnc,
		Proto0UserFakeDomain:      userconf.Proto0FakeDomain,
		Proto0SecretRouterEnc:     proto0SecretRouterEnc,
		Proto0SecretShufflerEnc:   proto0SecreShufflerEnc,
		Person:                    person,
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
			CountersOvc: DateSummaryNetCounters{
				Ver: DateSummaryNetCountersVersion,
			},
			CountersOutline: DateSummaryNetCounters{
				Ver: DateSummaryNetCountersVersion,
			},
			CountersProto0: DateSummaryNetCounters{
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

	userconf.FreeSlots = db.MaxUsers - len(data.Users)
	userconf.TotalSlots = db.MaxUsers

	if err := commitBrigade(f, data); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	if isBrigadier {
		fmt.Fprintf(os.Stderr, "Brigadier %s (%s) added\n", userconf.ID, base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub))

		return userconf, nil
	}

	fmt.Fprintf(os.Stderr, "User %s (%s) added\n", userconf.ID, base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub))

	return userconf, nil
}

var ErrUnresolvableCollision = fmt.Errorf("unresolvable collision")

func assembleUser(uid uuid.UUID, data *Brigade, fullname string, isBrigadier bool, maxUsers int) (uuid.UUID, netip.Addr, netip.Addr, string, error) {
	var ipv4, ipv6 netip.Addr

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

	switch uid {
	case uuid.Nil:
		for {
			id := uuid.New()
			if _, ok := idL[id.String()]; !ok {
				uid = id

				break
			}
		}
	default:
		if _, ok := idL[uid.String()]; ok {
			return uid, ipv4, ipv6, "", ErrUnresolvableCollision
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
func (db *BrigadeStorage) DeleteUser(id string, brigadier bool, onlyBlock bool) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	var (
		user *User
		idx  int
	)

	for i, u := range data.Users {
		if u.UserID.String() == id && u.IsBrigadier == brigadier {
			user = u
			idx = i

			break
		}
	}

	if user == nil {
		return ErrUserNotFound
	}

	if !user.IsBlocked {
		// if we catch a slowdown problems we need organize queue
		if err := vpnapi.WgPeerDel(data.BrigadeID, db.actualAddrPort, db.calculatedAddrPort, user.WgPublicKey, data.WgPublicKey); err != nil {
			return fmt.Errorf("peer del: %w", err)
		}

		if onlyBlock {
			user.IsBlocked = true
			user.BlockedAt = time.Now().UTC()
		}
	}

	if !onlyBlock {
		data.Users = append(data.Users[:idx], data.Users[idx+1:]...)
	}

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	fmt.Fprintf(os.Stderr, "User %s (%s) removed (only block: %v)\n", id, base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(user.WgPublicKey), onlyBlock)

	return nil
}

// UnblockUser - remove user from the storage.
func (db *BrigadeStorage) UnblockUser(id string) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	wgPub := []byte{}
	for _, user := range data.Users {
		if user.UserID.String() == id {
			wgPub = user.WgPublicKey

			if !user.IsBlocked {
				fmt.Fprintf(os.Stderr, "User %s (%s) already unblocked\n", id, base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub))

				break
			}

			kd6 := netip.Addr{}
			if user.IsBrigadier {
				kd6 = data.KeydeskIPv6
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

			user.IsBlocked = false
			user.BlockedAt = time.Time{}

			break
		}
	}

	if len(wgPub) == 0 {
		return ErrUserNotFound
	}

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	fmt.Fprintf(os.Stderr, "User %s (%s) unblocked\n", id, base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub))

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
			err := vpnapi.WgPeerDel(data.BrigadeID, db.actualAddrPort, db.calculatedAddrPort, wgPub, data.WgPublicKey)
			if err != nil {
				return "", namesgenerator.Person{}, fmt.Errorf("peer del: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Brigadier %s (%s) removed\n", user.UserID, base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub))

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

func (db *BrigadeStorage) GetUsersStats() (StatsCountersStack, int, int, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return StatsCountersStack{}, 0, 0, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	return data.StatsCountersStack, db.MaxUsers, db.MaxUsers - len(data.Users), nil
}
