package user

import (
	"fmt"
	"net/netip"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/wordsgens/namesgenerator"
)

type set[T comparable] map[T]struct{}

func getExisting(brigade *storage.Brigade) (set[string], set[uuid.UUID], set[netip.Addr], set[netip.Addr]) {
	name := make(set[string])
	uid := make(set[uuid.UUID])
	addr4 := map[netip.Addr]struct{}{
		brigade.IPv4CGNAT.Addr():                {},
		kdlib.LastPrefixIPv4(brigade.IPv4CGNAT): {},
	}
	addr6 := map[netip.Addr]struct{}{
		brigade.IPv6ULA.Addr():                {},
		kdlib.LastPrefixIPv6(brigade.IPv6ULA): {},
	}

	for _, user := range brigade.Users {
		name[user.Name] = struct{}{}
		uid[user.UserID] = struct{}{}
		addr4[user.IPv4Addr] = struct{}{}
		addr6[user.IPv6Addr] = struct{}{}
	}

	return name, uid, addr4, addr6
}

func getUniquePerson(nameSet set[string]) (string, namesgenerator.Person, error) {
	for {
		name, person, err := namesgenerator.ChemistryAwardeeShort()
		if err != nil {
			return "", person, fmt.Errorf("generate person: %w", err)
		}
		if _, ok := nameSet[name]; !ok {
			return name, person, err
		}
	}
}

func getUniqueUUID(uid set[uuid.UUID]) uuid.UUID {
	for {
		id := uuid.New()
		if _, ok := uid[id]; !ok {
			return id
		}
	}
}

func getUniqueAddr4(ip4CGNAT netip.Prefix, addr4 set[netip.Addr]) netip.Addr {
	for {
		addr := kdlib.RandomAddrIPv4(ip4CGNAT)
		if _, ok := addr4[addr]; !ok {
			return addr
		}
	}
}

func getUniqueAddr6(ip6ULA netip.Prefix, addr6 set[netip.Addr]) netip.Addr {
	for {
		addr := kdlib.RandomAddrIPv6(ip6ULA)
		if _, ok := addr6[addr]; !ok {
			return addr
		}
	}
}

func blurIP4(name, brigadeID string, ip4 netip.Addr, ip4CGNAT netip.Prefix) string {
	return fmt.Sprintf(
		"%03d %s",
		kdlib.BlurIpv4Addr(ip4, ip4CGNAT.Bits(), kdlib.ExtractUint32Salt(brigadeID)),
		name,
	)
}
