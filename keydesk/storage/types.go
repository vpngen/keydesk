package storage

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// UsersNetworks - nets from stats.
type UsersNetworks map[string]time.Time

// DateSummaryNetCountersVersion - json version.
const DateSummaryNetCountersVersion = 1

// DateSummaryNetCounters - traffic counters container.
type DateSummaryNetCounters struct {
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

// UserCounters - user counters.
type UsersCounters struct {
	TotalUsersCount       int `json:"total_users_count"`
	ActiveUsersCount      int `json:"active_users_count"`
	ActiveWgUsersCount    int `json:"active_wg_users_count"`
	ActiveIPSecUsersCount int `json:"active_ipsec_users_count"`
	ActiveOvcUsersCount   int `json:"active_ovc_users_count"`
	ThrottledUsersCount   int `json:"throttled_users_count"`
}

// NetCounters - net counters.
type NetCounters struct {
	TotalTraffic      RxTx `json:"total_traffic"`
	TotalWgTraffic    RxTx `json:"total_wg_traffic"`
	TotalIPSecTraffic RxTx `json:"total_ipsec_traffic"`
	TotalOvcTraffic   RxTx `json:"total_ovc_traffic"`
}

// BrigadeCounters - brigade counters.
type BrigadeCounters struct {
	UsersCounters
	TotalTraffic       DateSummaryNetCounters `json:"total_traffic"`
	TotalWgTraffic     DateSummaryNetCounters `json:"total_wg_traffic"`
	TotalIPSecTraffic  DateSummaryNetCounters `json:"total_ipsec_traffic"`
	TotalOvcTraffic    DateSummaryNetCounters `json:"total_ovc_traffic"`
	CountersUpdateTime time.Time              `json:"counters_update_time"`
}

type TrafficCountersContainer struct {
	TrafficSummary RxTx
	TrafficWg      RxTx
	TrafficIPSec   RxTx
	TrafficOvc     RxTx
}

type StatsCounters struct {
	UsersCounters
	NetCounters
	CountersUpdateTime time.Time `json:"counters_update_time"`
}

// StatsCountersStack - counters month based stack.
type StatsCountersStack [12]StatsCounters

// Put - put counters to stack. If month changed, then shift stack.
func (x *StatsCountersStack) Put(counters BrigadeCounters, traffic TrafficCountersContainer) {
	now := counters.CountersUpdateTime
	last := x[len(x)-1].CountersUpdateTime

	if !last.IsZero() && (last.Year() != now.Year() || last.Month() != now.Month()) {
		for i := 0; i < len(x)-1; i++ {
			x[i] = x[i+1]
		}

		x[len(x)-1] = StatsCounters{}
	}

	stats := &x[len(x)-1]
	stats.UsersCounters = counters.UsersCounters
	stats.NetCounters.TotalTraffic.Inc(traffic.TrafficSummary.Rx, traffic.TrafficSummary.Tx)
	stats.NetCounters.TotalWgTraffic.Inc(traffic.TrafficWg.Rx, traffic.TrafficWg.Tx)
	stats.NetCounters.TotalIPSecTraffic.Inc(traffic.TrafficIPSec.Rx, traffic.TrafficIPSec.Tx)
	stats.CountersUpdateTime = counters.CountersUpdateTime
}

// QuotaVesrion - json version.
const QuotaVesrion = 3

// Quota - user quota.
type Quota struct {
	Ver                   int                    `json:"version"`
	OSWgCounters          RxTx                   `json:"os_wg_counters"`
	OSIPSecCounters       RxTx                   `json:"os_ipsec_counters"`
	OSOvcCounters         RxTx                   `json:"os_ovc_counters"`
	CountersTotal         DateSummaryNetCounters `json:"counters_total"`
	CountersWg            DateSummaryNetCounters `json:"counters_wg"`
	CountersIPSec         DateSummaryNetCounters `json:"counters_ipsec"`
	CountersOvc           DateSummaryNetCounters `json:"counters_ovc"`
	LimitMonthlyRemaining uint64                 `json:"limit_monthly_remaining"`
	LimitMonthlyResetOn   time.Time              `json:"limit_monthly_reset_on,omitempty"`
	LastActivity          LastActivityPoints     `json:"last_activity,omitempty"`
	LastWgActivity        LastActivityPoints     `json:"last_wg_activity,omitempty"`
	LastIPSecActivity     LastActivityPoints     `json:"last_ipsec_activity,omitempty"`
	LastOvcActivity       LastActivityPoints     `json:"last_ovc_activity,omitempty"`
	ThrottlingTill        time.Time              `json:"throttling_till,omitempty"`
}

// UserVersion - json version.
const UserVersion = 4

// User - user structure.
type User struct {
	Ver                       int                   `json:"version"`
	UserID                    uuid.UUID             `json:"user_id"`
	Name                      string                `json:"name"`
	CreatedAt                 time.Time             `json:"created_at"`
	IsBrigadier               bool                  `json:"is_brigadier,omitempty"`
	IPv4Addr                  netip.Addr            `json:"ipv4_addr"`
	IPv6Addr                  netip.Addr            `json:"ipv6_addr"`
	WgPublicKey               []byte                `json:"wg_public_key"`
	WgPSKRouterEnc            []byte                `json:"wg_psk_router_enc"`
	WgPSKShufflerEnc          []byte                `json:"wg_psk_shuffler_enc"`
	CloakByPassUIDRouterEnc   string                `json:"cloak_bypass_uid_router_enc"`   // Cloak bypass UID for router prepared
	CloakByPassUIDShufflerEnc string                `json:"cloak_bypass_uid_shuffler_enc"` // Cloak bypass UID for shuffler prepared
	OvCSRGzipBase64           string                `json:"openvpn_csr,omitempty"`         // OpenVPN CSR base64 encoded
	Person                    namesgenerator.Person `json:"person"`
	Quotas                    Quota                 `json:"quotas"`
}

// BrigadeVersion - json version.
const BrigadeVersion = 8

// Brigade - brigade.
type Brigade struct {
	BrigadeCounters
	StatsCountersStack    `json:"counters_stack"`
	Ver                   int           `json:"version"`
	BrigadeID             string        `json:"brigade_id"`
	CreatedAt             time.Time     `json:"created_at"`
	WgPublicKey           []byte        `json:"wg_public_key"`
	WgPrivateRouterEnc    []byte        `json:"wg_private_router_enc"`
	WgPrivateShufflerEnc  []byte        `json:"wg_private_shuffler_enc"`
	CloakFakeDomain       string        `json:"cloak_faek_domain"`           // Cloak fake domain
	OvCAKeyRouterEnc      string        `json:"openvpn_ca_key_router_enc"`   // OpenVPN CA key PEM PKSC8 for router prepared
	OvCAKeyShufflerEnc    string        `json:"openvpn_ca_key_shuffler_enc"` // OpenVPN CA key PEM PKSC8 for shuffler prepared
	OvCACertPemGzipBase64 string        `json:"openvpn_ca_cert"`             // OpenVPN CA cert PEM encoded
	EndpointIPv4          netip.Addr    `json:"endpoint_ipv4"`
	EndpointDomain        string        `json:"endpoint_domain"`
	EndpointPort          uint16        `json:"endpoint_port"`
	DNSv4                 netip.Addr    `json:"dns4"`
	DNSv6                 netip.Addr    `json:"dns6"`
	KeydeskIPv6           netip.Addr    `json:"keydesk_ipv6"`
	IPv4CGNAT             netip.Prefix  `json:"ipv4_cgnat"`
	IPv6ULA               netip.Prefix  `json:"ipv6_ula"`
	KeydeskFirstVisit     time.Time     `json:"keydesk_first_visit,omitempty"`
	Users                 []*User       `json:"users,omitempty"`
	Endpoints             UsersNetworks `json:"endpoints,omitempty"`
}

// UserConfig - new user structure.
type UserConfig struct {
	ID               uuid.UUID
	Name             string
	EndpointWgPublic []byte
	DNSv4, DNSv6     netip.Addr
	IPv4, IPv6       netip.Addr
	EndpointIPv4     netip.Addr
	EndpointDomain   string
	EndPointPort     uint16
	OvCACertPem      string
	OvClientCertPem  string
	CloakByPassUID   []byte
	CloakFakeDomain  string
}

// BrigadeConfig - new brigade structure.
type BrigadeConfig struct {
	BrigadeID      string
	EndpointIPv4   netip.Addr
	EndpointDomain string
	EndPointPort   uint16
	DNSIPv4        netip.Addr
	DNSIPv6        netip.Addr
	IPv4CGNAT      netip.Prefix
	IPv6ULA        netip.Prefix
	KeydeskIPv6    netip.Addr
}

// StatsVersion - json version.
const StatsVersion = 2

// Stats - statistics.
type Stats struct {
	StatsCounters
	Ver               int           `json:"version"`
	BrigadeID         string        `json:"brigade_id"`
	UpdateTime        time.Time     `json:"update_time"`
	BrigadeCreatedAt  time.Time     `json:"brigade_created_at"`
	KeydeskFirstVisit time.Time     `json:"keydesk_first_visit,omitempty"`
	Endpoints         UsersNetworks `json:"endpoints,omitempty"`
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
