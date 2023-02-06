package keydesk

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// Quota - user quota.
type Quota struct {
	OSCounterMtime        time.Time `json:"os_counter_mtime"`
	OSCounterRX           uint64    `json:"os_counter_rx"`
	OSCounterTX           uint64    `json:"os_counter_tx"`
	LimitMonthlyRemaining uint64    `json:"limit_monthly_remaining"`
	LimitMonthlyResetOn   time.Time `json:"limit_monthly_reset_on"`
	LastActivity          time.Time `json:"last_activity"`
	ThrottlingTill        time.Time `json:"throttling_till"`
}

// User - user structure.
type User struct {
	UserID           uuid.UUID             `json:"user_id"`
	Name             string                `json:"name"`
	CreatedAt        time.Time             `json:"created_at"`
	IsBrigadier      bool                  `json:"is_brigadier,omitempty"`
	IPv4Addr         netip.Addr            `json:"ipv4_addr"`
	IPv6Addr         netip.Addr            `json:"ipv6_addr"`
	WgPublicKey      []byte                `json:"wg_public_key"`
	WgPSKRouterEnc   []byte                `json:"wg_psk_router_enc"`
	WgPSKShufflerEnc []byte                `json:"wg_psk_shuffler_enc"`
	Person           namesgenerator.Person `json:"person"`
	Quota            Quota                 `json:"quota"`
}

// Brigade - brigade.
type Brigade struct {
	BrigadeID            string       `json:"brigade_id"`
	CreatedAt            time.Time    `json:"created_at"`
	KeydeskLastVisit     time.Time    `json:"keydesk_last_visit,omitempty"`
	WgPublicKey          []byte       `json:"wg_public_key"`
	WgPrivateRouterEnc   []byte       `json:"wg_private_router_enc"`
	WgPrivateShufflerEnc []byte       `json:"wg_private_shuffler_enc"`
	EndpointIPv4         netip.Addr   `json:"endpoint_ipv4"`
	DNSv4                netip.Addr   `json:"dns4"`
	DNSv6                netip.Addr   `json:"dns6"`
	KeydeskIPv6          netip.Addr   `json:"keydesk_ipv6"`
	IPv4CGNAT            netip.Prefix `json:"ipv4_cgnat"`
	IPv6ULA              netip.Prefix `json:"ipv6_ula"`
	Users                []User       `json:"users,omitempty"`
}

// UserConfig - new user structure.
type UserConfig struct {
	ID               uuid.UUID
	Name             string
	EndpointWgPublic []byte
	DNSv4, DNSv6     netip.Addr
	IPv4, IPv6       netip.Addr
	EndpointIPv4     netip.Addr
}
