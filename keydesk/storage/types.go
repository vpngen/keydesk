package storage

import (
	"net/netip"
	"time"

	"github.com/SherClockHolmes/webpush-go"

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
	TotalUsersCount         int `json:"total_users_count"`
	ActiveUsersCount        int `json:"active_users_count"`
	ActiveWgUsersCount      int `json:"active_wg_users_count"`
	ActiveIPSecUsersCount   int `json:"active_ipsec_users_count"`
	ActiveOvcUsersCount     int `json:"active_ovc_users_count"`
	ActiveOutlineUsersCount int `json:"active_outline_users_count"`
	ThrottledUsersCount     int `json:"throttled_users_count"`
}

// NetCounters - net counters.
type NetCounters struct {
	TotalTraffic        RxTx `json:"total_traffic"`
	TotalWgTraffic      RxTx `json:"total_wg_traffic"`
	TotalIPSecTraffic   RxTx `json:"total_ipsec_traffic"`
	TotalOvcTraffic     RxTx `json:"total_ovc_traffic"`
	TotalOutlineTraffic RxTx `json:"total_outline_traffic"`
}

// BrigadeCounters - brigade counters.
type BrigadeCounters struct {
	UsersCounters
	TotalTraffic        DateSummaryNetCounters `json:"total_traffic"`
	TotalWgTraffic      DateSummaryNetCounters `json:"total_wg_traffic"`
	TotalIPSecTraffic   DateSummaryNetCounters `json:"total_ipsec_traffic"`
	TotalOvcTraffic     DateSummaryNetCounters `json:"total_ovc_traffic"`
	TotalOutlineTraffic DateSummaryNetCounters `json:"total_outline_traffic"`
	CountersUpdateTime  time.Time              `json:"counters_update_time"`
}

type TrafficCountersContainer struct {
	TrafficSummary RxTx
	TrafficWg      RxTx
	TrafficIPSec   RxTx
	TrafficOvc     RxTx
	TrafficOutline RxTx
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
	stats.NetCounters.TotalOvcTraffic.Inc(traffic.TrafficOvc.Rx, traffic.TrafficOvc.Tx)
	stats.NetCounters.TotalOutlineTraffic.Inc(traffic.TrafficOutline.Rx, traffic.TrafficOutline.Tx)
	stats.CountersUpdateTime = counters.CountersUpdateTime
}

// QuotaVesrion - json version.
const QuotaVesrion = 5

// Quota - user quota.
type Quota struct {
	Ver                   int                    `json:"version"`
	OSWgCounters          RxTx                   `json:"os_wg_counters"`
	OSIPSecCounters       RxTx                   `json:"os_ipsec_counters"`
	OSOvcCounters         RxTx                   `json:"os_ovc_counters"`
	OSOutlineCounters     RxTx                   `json:"os_outline_counters"`
	CountersTotal         DateSummaryNetCounters `json:"counters_total"`
	CountersWg            DateSummaryNetCounters `json:"counters_wg"`
	CountersIPSec         DateSummaryNetCounters `json:"counters_ipsec"`
	CountersOvc           DateSummaryNetCounters `json:"counters_ovc"`
	CountersOutline       DateSummaryNetCounters `json:"counters_outline"`
	LimitMonthlyRemaining uint64                 `json:"limit_monthly_remaining"`
	LimitMonthlyResetOn   time.Time              `json:"limit_monthly_reset_on,omitempty"`
	LastActivity          LastActivityPoints     `json:"last_activity,omitempty"`
	LastWgActivity        LastActivityPoints     `json:"last_wg_activity,omitempty"`
	LastIPSecActivity     LastActivityPoints     `json:"last_ipsec_activity,omitempty"`
	LastOvcActivity       LastActivityPoints     `json:"last_ovc_activity,omitempty"`
	LastOutlineActivity   LastActivityPoints     `json:"last_outline_activity,omitempty"`
	ThrottlingTill        time.Time              `json:"throttling_till,omitempty"`
}

// UserVersion - json version.
const UserVersion = 5

// User - user structure.
type User struct {
	Ver                       int                   `json:"version"`
	UserID                    uuid.UUID             `json:"user_id"`
	Name                      string                `json:"name"`
	CreatedAt                 time.Time             `json:"created_at"`
	IsBrigadier               bool                  `json:"is_brigadier,omitempty"`
	IPv4Addr                  netip.Addr            `json:"ipv4_addr"`
	IPv6Addr                  netip.Addr            `json:"ipv6_addr"`
	EndpointDomain            string                `json:"endpoint_domain,omitempty"`
	WgPublicKey               []byte                `json:"wg_public_key"`
	WgPSKRouterEnc            []byte                `json:"wg_psk_router_enc"`
	WgPSKShufflerEnc          []byte                `json:"wg_psk_shuffler_enc"`
	CloakByPassUIDRouterEnc   string                `json:"cloak_bypass_uid_router_enc"`   // Cloak bypass UID for router prepared
	CloakByPassUIDShufflerEnc string                `json:"cloak_bypass_uid_shuffler_enc"` // Cloak bypass UID for shuffler prepared
	OvCSRGzipBase64           string                `json:"openvpn_csr,omitempty"`         // OpenVPN CSR base64 encoded
	IPSecUsernameRouterEnc    string                `json:"ipsec_username_router_enc"`     // IPSec user name for router prepared
	IPSecUsernameShufflerEnc  string                `json:"ipsec_username_shuffler_enc"`   // IPSec user name for shuffler prepared
	IPSecPasswordRouterEnc    string                `json:"ipsec_password_router_enc"`     // IPSec password for router prepared
	IPSecPasswordShufflerEnc  string                `json:"ipsec_password_shuffler_enc"`   // IPSec password for shuffler prepared
	OutlineSecretRouterEnc    string                `json:"outline_secret_router_enc"`     // Outline secret for router prepared
	OutlineSecretShufflerEnc  string                `json:"outline_secret_shuffler_enc"`   // Outline secret for shuffler prepared
	Person                    namesgenerator.Person `json:"person"`
	Quotas                    Quota                 `json:"quotas"`
}

func NewUser(userID uuid.UUID, name string, createdAt time.Time, isBrigadier bool, IPv4Addr netip.Addr, IPv6Addr netip.Addr, person namesgenerator.Person) User {
	return User{UserID: userID, Name: name, CreatedAt: createdAt, IsBrigadier: isBrigadier, IPv4Addr: IPv4Addr, IPv6Addr: IPv6Addr, Person: person}
}

// BrigadeVersion - json version.
const BrigadeVersion = 9

type Mode = string

const (
	ModeBrigade  Mode = "brigade"
	ModeShuffler Mode = "shuffler"
	MaxUsers          = uint(255)
)

// Brigade - brigade.
type Brigade struct {
	BrigadeCounters
	StatsCountersStack    `json:"counters_stack"`
	Ver                   int                  `json:"version"`
	BrigadeID             string               `json:"brigade_id"`
	CreatedAt             time.Time            `json:"created_at"`
	Mode                  Mode                 `json:"mode"`
	MaxUsers              uint                 `json:"max_users,omitempty"`
	WgPublicKey           []byte               `json:"wg_public_key"`
	WgPrivateRouterEnc    []byte               `json:"wg_private_router_enc"`
	WgPrivateShufflerEnc  []byte               `json:"wg_private_shuffler_enc"`
	CloakFakeDomain       string               `json:"cloak_fake_domain"`           // Cloak fake domain
	CloakFaekDomain       string               `json:"cloak_faek_domain"`           // Cloak fake domain
	OvCAKeyRouterEnc      string               `json:"openvpn_ca_key_router_enc"`   // OpenVPN CA key PEM PKSC8 for router prepared
	OvCAKeyShufflerEnc    string               `json:"openvpn_ca_key_shuffler_enc"` // OpenVPN CA key PEM PKSC8 for shuffler prepared
	OvCACertPemGzipBase64 string               `json:"openvpn_ca_cert"`             // OpenVPN CA cert PEM encoded
	IPSecPSK              string               `json:"ipsec_psk"`                   // IPSec PSK
	IPSecPSKRouterEnc     string               `json:"ipsec_psk_router_enc"`        // IPSec PSK for router prepared
	IPSecPSKShufflerEnc   string               `json:"ipsec_psk_shuffler_enc"`      // IPSec PSK for shuffler prepared
	EndpointIPv4          netip.Addr           `json:"endpoint_ipv4"`
	EndpointDomain        string               `json:"endpoint_domain"`
	EndpointPort          uint16               `json:"endpoint_port"`
	OutlinePort           uint16               `json:"outline_port"`
	DNSv4                 netip.Addr           `json:"dns4"`
	DNSv6                 netip.Addr           `json:"dns6"`
	KeydeskIPv6           netip.Addr           `json:"keydesk_ipv6"`
	IPv4CGNAT             netip.Prefix         `json:"ipv4_cgnat"`
	IPv6ULA               netip.Prefix         `json:"ipv6_ula"`
	KeydeskFirstVisit     time.Time            `json:"keydesk_first_visit,omitempty"`
	Users                 []*User              `json:"users,omitempty"`
	Endpoints             UsersNetworks        `json:"endpoints,omitempty"`
	Messages              []Message            `json:"messages,omitempty"`
	Subscription          webpush.Subscription `json:"subscription"`
}

func (b Brigade) GetSupportedVPNProtocols() []string {
	protocols := []string{"wg"} // wg is always supported

	if b.OvCACertPemGzipBase64 != "" && b.OvCAKeyRouterEnc != "" && b.OvCAKeyShufflerEnc != "" {
		protocols = append(protocols, "ovc")
	}

	if b.IPSecPSK != "" && b.IPSecPSKRouterEnc != "" && b.IPSecPSKShufflerEnc != "" {
		protocols = append(protocols, "ipsec")
	}

	if b.OutlinePort > 0 {
		protocols = append(protocols, "outline")
	}

	return protocols
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
	EndpointPort     uint16
	OvCACertPem      string
	OvClientCertPem  string
	CloakByPassUID   []byte
	CloakFakeDomain  string
	IPSecPSK         string
	IPSecUserName    string
	IPSecPassword    string
	OutlinePort      uint16
}

// BrigadeConfig - new brigade structure.
type BrigadeConfig struct {
	BrigadeID      string
	EndpointIPv4   netip.Addr
	EndpointDomain string
	EndpointPort   uint16
	OutlinePort    uint16
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

type Message struct {
	ID        uuid.UUID     `json:"id"`
	Text      string        `json:"text"`
	Title     string        `json:"title"`
	IsRead    bool          `json:"is_read"`
	Priority  int           `json:"priority"`
	CreatedAt time.Time     `json:"created_at"`
	TTL       time.Duration `json:"ttl,omitempty"`
}

type Keys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}
