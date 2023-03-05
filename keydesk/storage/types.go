package storage

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// NetCountersVersion - json version.
const NetCountersVersion = 1

// NetCounters - traffic counters container.
type NetCounters struct {
	Update  time.Time `json:"update"`
	Total   RxTx      `json:"total"`
	Monthly RxTx      `json:"monthly"`
	Weekly  RxTx      `json:"weekly"`
	Daily   RxTx      `json:"daily"`
	Ver     int       `json:"version"`
}

// RxTx - rx/tx counters.
type RxTx struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
}

// QuotaVesrion - json version.
const QuotaVesrion = 1

// Quota - user quota.
type Quota struct {
	UserID                uuid.UUID   `json:"user_id"`
	Counters              NetCounters `json:"counters"`
	LimitMonthlyRemaining uint64      `json:"limit_monthly_remaining"`
	LimitMonthlyResetOn   time.Time   `json:"limit_monthly_reset_on"`
	LastActivity          time.Time   `json:"last_activity"`
	ThrottlingTill        time.Time   `json:"throttling_till"`
	Ver                   int         `json:"version"`
}

// UsersQuotas - list users quotas.
type UsersQuotas struct {
	BrigadeID string           `json:"brigade_id"`
	Total     NetCounters      `json:"total"`
	Users     map[string]Quota `json:"users,omitempty"`
	Ver       int              `json:"version"`
}

// UserVersion - json version.
const UserVersion = 1

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
	Ver              int                   `json:"version"`
}

// KeydeskCountersVersion - json version.
const KeydeskCountersVersion = 1

// KeydeskCounters - counters.
type KeydeskCounters struct {
	BrigadeID        string    `json:"brigade_id"`
	KeydeskLastVisit time.Time `json:"keydesk_last_visit,omitempty"`
	Ver              int       `json:"version"`
}

// BrigadeVersion - json version.
const BrigadeVersion = 1

// Brigade - brigade.
type Brigade struct {
	BrigadeID            string       `json:"brigade_id"`
	CreatedAt            time.Time    `json:"created_at"`
	WgPublicKey          []byte       `json:"wg_public_key"`
	WgPrivateRouterEnc   []byte       `json:"wg_private_router_enc"`
	WgPrivateShufflerEnc []byte       `json:"wg_private_shuffler_enc"`
	EndpointIPv4         netip.Addr   `json:"endpoint_ipv4"`
	DNSv4                netip.Addr   `json:"dns4"`
	DNSv6                netip.Addr   `json:"dns6"`
	KeydeskIPv6          netip.Addr   `json:"keydesk_ipv6"`
	IPv4CGNAT            netip.Prefix `json:"ipv4_cgnat"`
	IPv6ULA              netip.Prefix `json:"ipv6_ula"`
	Users                []*User      `json:"users,omitempty"`
	Ver                  int          `json:"version"`
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

// BrigadeConfig - new brigade structure.
type BrigadeConfig struct {
	BrigadeID    string
	EndpointIPv4 netip.Addr
	DNSIPv4      netip.Addr
	DNSIPv6      netip.Addr
	IPv4CGNAT    netip.Prefix
	IPv6ULA      netip.Prefix
	KeydeskIPv6  netip.Addr
}

// StatsVersion - json version.
const StatsVersion = 1

// Stats - statistics.
type Stats struct {
	BrigadeID          string    `json:"brigade_id"`
	Updated            time.Time `json:"updated"`
	BrigadeCreatedAt   time.Time `json:"brigade_created_at"`
	KeydeskLastVisit   time.Time `json:"keydesk_last_visit,omitempty"`
	UsersCount         int       `json:"users_count"`
	ActiveUsersCount   int       `json:"active_users_count"`
	ThrottledUserCount int       `json:"throttled_users_count"`
	TotalRx            uint64    `json:"total_rx"`
	TotalTx            uint64    `json:"total_tx"`
	Ver                int       `json:"version"`
}
