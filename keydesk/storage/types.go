package storage

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// UsersNetworks - nets from stats.
type UsersNetworks map[string]time.Time

// NetCountersVersion - json version.
const NetCountersVersion = 1

// NetCounters - traffic counters container.
type NetCounters struct {
	Ver     int       `json:"version"`
	Update  time.Time `json:"update,omitempty"`
	Total   RxTx      `json:"total"`
	Yearly  RxTx      `json:"yearly"`
	Monthly RxTx      `json:"monthly"`
	Weekly  RxTx      `json:"weekly"`
	Daily   RxTx      `json:"daily"`
}

// RxTx - rx/tx counters.
type RxTx struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
}

// Inc - increment counters.
func (x *RxTx) Inc(rx, tx uint64) {
	x.Rx += rx
	x.Tx += tx
}

// Reset - reset counters.
func (x *RxTx) Reset(rx, tx uint64) {
	x.Rx = rx
	x.Tx = tx
}

// BaseBrigadeCounters - brigade counters.
type BaseBrigadeCounters struct {
	TotalUsersCount       int         `json:"total_users_count"`
	ActiveUsersCount      int         `json:"active_users_count"`
	ActiveWgUsersCount    int         `json:"active_wg_users_count"`
	ActiveIPSecUsersCount int         `json:"active_ipsec_users_count"`
	ThrottledUsersCount   int         `json:"throttled_users_count"`
	Total                 NetCounters `json:"total"`
	TotalWg               NetCounters `json:"total_wg"`
	TotalIPSec            NetCounters `json:"total_ipsec"`
	TotalTraffic          NetCounters `json:"total_traffic"`
	TotalWgTraffic        NetCounters `json:"total_wg_traffic"`
	TotalIPSecTraffic     NetCounters `json:"total_ipsec_traffic"`
	CountersUpdateTime    time.Time   `json:"counters_update_time"`
}

// BrigadeCountersStack - brigade counters year based stack.
type BrigadeCountersStack [12]BaseBrigadeCounters

// Put - put counters to stack. If month changed, then shift stack.
func (x *BrigadeCountersStack) Put(counters BaseBrigadeCounters) {
	now := counters.CountersUpdateTime
	last := x[len(x)-1].CountersUpdateTime

	if !last.IsZero() && (last.Year() != now.Year() || last.Month() != now.Month()) {
		for i := 0; i < len(x)-1; i++ {
			x[i] = x[i+1]
		}

		x[len(x)-1] = counters
	}

	x[len(x)-1] = counters
}

// QuotaVesrion - json version.
const QuotaVesrion = 2

// Quota - user quota.
type Quota struct {
	Ver                   int                `json:"version"`
	OSWgCounters          RxTx               `json:"os_wg_counters"`
	OSIPSecCounters       RxTx               `json:"os_ipsec_counters"`
	CountersTotal         NetCounters        `json:"counters_total"`
	CountersWg            NetCounters        `json:"counters_wg"`
	CountersIPSec         NetCounters        `json:"counters_ipsec"`
	LimitMonthlyRemaining uint64             `json:"limit_monthly_remaining"`
	LimitMonthlyResetOn   time.Time          `json:"limit_monthly_reset_on,omitempty"`
	LastActivity          LastActivityPoints `json:"last_activity,omitempty"`
	LastWgActivity        LastActivityPoints `json:"last_wg_activity,omitempty"`
	LastIPSecActivity     LastActivityPoints `json:"last_ipsec_activity,omitempty"`
	ThrottlingTill        time.Time          `json:"throttling_till,omitempty"`
}

// UserVersion - json version.
const UserVersion = 2

// User - user structure.
type User struct {
	Ver              int                   `json:"version"`
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
	Quotas           Quota                 `json:"quotas"`
}

// BrigadeVersion - json version.
const BrigadeVersion = 4

// Brigade - brigade.
type Brigade struct {
	BaseBrigadeCounters
	BrigadeCountersStack `json:"counters_year_stack"`
	Ver                  int           `json:"version"`
	BrigadeID            string        `json:"brigade_id"`
	CreatedAt            time.Time     `json:"created_at"`
	WgPublicKey          []byte        `json:"wg_public_key"`
	WgPrivateRouterEnc   []byte        `json:"wg_private_router_enc"`
	WgPrivateShufflerEnc []byte        `json:"wg_private_shuffler_enc"`
	EndpointIPv4         netip.Addr    `json:"endpoint_ipv4"`
	DNSv4                netip.Addr    `json:"dns4"`
	DNSv6                netip.Addr    `json:"dns6"`
	KeydeskIPv6          netip.Addr    `json:"keydesk_ipv6"`
	IPv4CGNAT            netip.Prefix  `json:"ipv4_cgnat"`
	IPv6ULA              netip.Prefix  `json:"ipv6_ula"`
	KeydeskLastVisit     time.Time     `json:"keydesk_last_visit,omitempty"`
	Users                []*User       `json:"users,omitempty"`
	Endpoints            UsersNetworks `json:"endpoints,omitempty"`
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
const StatsVersion = 2

// Stats - statistics.
type Stats struct {
	BaseBrigadeCounters
	Ver              int           `json:"version"`
	BrigadeID        string        `json:"brigade_id"`
	UpdateTime       time.Time     `json:"update_time"`
	BrigadeCreatedAt time.Time     `json:"brigade_created_at"`
	KeydeskLastVisit time.Time     `json:"keydesk_last_visit,omitempty"`
	Endpoints        UsersNetworks `json:"endpoints,omitempty"`
}

// LastActivityPoints - traffic counters container.
type LastActivityPoints struct {
	Update      time.Time `json:"update,omitempty"`
	Total       time.Time `json:"total"`
	Yearly      time.Time `json:"yearly"`
	Monthly     time.Time `json:"monthly"`
	PrevMonthly time.Time `json:"prev_monthly"`
	Weekly      time.Time `json:"weekly"`
	Daily       time.Time `json:"daily"`
}
