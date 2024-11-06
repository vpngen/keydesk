package vpnapi

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

const (
	wgStatName      = "wireguard"
	ipsecStatName   = "ipsec"
	ovcStatName     = "cloak-openvpn"
	olcStatName     = "cloak-ss"
	outlineStatName = "outline-ss"
	proto0StatName  = "proto0"
)

// WgStatTimestamp - VPN stat timestamp.
type WgStatTimestamp struct {
	Timestamp int64
	Time      time.Time
}

// WgStatTraffic - VPN stat traffic.
type WgStatTraffic struct {
	Rx uint64
	Tx uint64
}

// WgStatTrafficMap - VPN stat traffic map, key is User wg_public_key.
// Dedicated map objects for wg and ipsec.
type WgStatTrafficMap struct {
	Wg      map[string]*WgStatTraffic
	IPSec   map[string]*WgStatTraffic
	Ovc     map[string]*WgStatTraffic
	Outline map[string]*WgStatTraffic
	Proto0  map[string]*WgStatTraffic
}

// WgStatLastActivityMap - VPN stat last activity map, key is User wg_public_key.
// Dedicated map objects for wg and ipsec.
type WgStatLastActivityMap struct {
	Wg      map[string]time.Time
	IPSec   map[string]time.Time
	Ovc     map[string]time.Time
	Olc     map[string]time.Time
	Outline map[string]time.Time
	Proto0  map[string]time.Time
}

// WgStatEndpointMap - VPN stat endpoint map, key is User wg_public_key.
// Dedicated map objects for wg and ipsec.
type WgStatEndpointMap struct {
	Wg      map[string]netip.Prefix
	IPSec   map[string]netip.Prefix
	Ovc     map[string]netip.Prefix
	Olc     map[string]netip.Prefix
	Outline map[string]netip.Prefix
	Proto0  map[string]netip.Prefix
}

// ErrInvalidStatFormat - invalid stat format.
var ErrInvalidStatFormat = errors.New("invalid stat")

// NewWgStatTrafficMap - create new WgStatTrafficMap.
func NewWgStatTrafficMap() *WgStatTrafficMap {
	return &WgStatTrafficMap{
		Wg:      make(map[string]*WgStatTraffic),
		IPSec:   make(map[string]*WgStatTraffic),
		Ovc:     make(map[string]*WgStatTraffic),
		Outline: make(map[string]*WgStatTraffic),
		Proto0:  make(map[string]*WgStatTraffic),
	}
}

// NewWgStatLastActivityMap - create new WgStatLastActivityMap.
func NewWgStatLastActivityMap() *WgStatLastActivityMap {
	return &WgStatLastActivityMap{
		Wg:      make(map[string]time.Time),
		IPSec:   make(map[string]time.Time),
		Ovc:     make(map[string]time.Time),
		Olc:     make(map[string]time.Time),
		Outline: make(map[string]time.Time),
		Proto0:  make(map[string]time.Time),
	}
}

// NewWgStatEndpointMap - create new WgStatEndpointMap.
func NewWgStatEndpointMap() *WgStatEndpointMap {
	return &WgStatEndpointMap{
		Wg:      make(map[string]netip.Prefix),
		IPSec:   make(map[string]netip.Prefix),
		Ovc:     make(map[string]netip.Prefix),
		Olc:     make(map[string]netip.Prefix),
		Outline: make(map[string]netip.Prefix),
		Proto0:  make(map[string]netip.Prefix),
	}
}

// WgStatParseTimestamp - parse timestamp value.
func WgStatParseTimestamp(timestamp string) (*WgStatTimestamp, error) {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return &WgStatTimestamp{
		Timestamp: ts,
		Time:      time.Unix(ts, 0).UTC(),
	}, nil
}

func WgStatParseTraffic(traffic WgStatTrafficMapIn) (*WgStatTrafficMap, error) {
	m := NewWgStatTrafficMap()

	for id, data := range traffic {
		id = strings.ReplaceAll(strings.ReplaceAll(id, "-", "+"), "_", "/")

		for vpnType, traffic := range data {
			rx, err := strconv.ParseUint(traffic.Received, 10, 64)
			if err != nil {
				continue
			}

			tx, err := strconv.ParseUint(traffic.Sent, 10, 64)
			if err != nil {
				continue
			}

			switch vpnType {
			case wgStatName:
				m.Wg[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			case ipsecStatName:
				m.IPSec[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			case ovcStatName:
				m.Ovc[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			case outlineStatName:
				m.Outline[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			case proto0StatName:
				m.Proto0[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			}
		}
	}

	return m, nil
}

func WgStatParseLastActivity(lastSeen WgStatLastseenMapIn) (*WgStatLastActivityMap, error) {
	m := NewWgStatLastActivityMap()

	for id, data := range lastSeen {
		id = strings.ReplaceAll(strings.ReplaceAll(id, "-", "+"), "_", "/")

		for vpnType, lastSeen := range data {
			ts, err := strconv.ParseInt(lastSeen.Timestamp, 10, 64)
			if err != nil {
				continue
			}

			if ts == 0 {
				continue
			}

			switch vpnType {
			case wgStatName:
				m.Wg[id] = time.Unix(ts, 0).UTC()
			case ipsecStatName:
				m.IPSec[id] = time.Unix(ts, 0).UTC()
			case ovcStatName:
				m.Ovc[id] = time.Unix(ts, 0).UTC()
			case olcStatName:
				m.Olc[id] = time.Unix(ts, 0).UTC()
			case outlineStatName:
				m.Outline[id] = time.Unix(ts, 0).UTC()
			case proto0StatName:
				m.Proto0[id] = time.Unix(ts, 0).UTC()
			}
		}
	}

	return m, nil
}

func WgStatParseEndpoints(endpoints WgStatEndpointMapIn) (*WgStatEndpointMap, error) {
	m := NewWgStatEndpointMap()

	for id, data := range endpoints {
		id = strings.ReplaceAll(strings.ReplaceAll(id, "-", "+"), "_", "/")

		for vpnType, endpoint := range data {
			if endpoint.Subnet == "(none)" {
				continue
			}

			prefix, err := netip.ParsePrefix(endpoint.Subnet)
			if err != nil {
				continue
			}

			if !prefix.IsValid() {
				continue
			}

			switch vpnType {
			case wgStatName:
				m.Wg[id] = prefix
			case ipsecStatName:
				m.IPSec[id] = prefix
			case ovcStatName:
				m.Ovc[id] = prefix
			case olcStatName:
				m.Olc[id] = prefix
			case outlineStatName:
				m.Outline[id] = prefix
			case proto0StatName:
				m.Proto0[id] = prefix
			}
		}
	}

	return m, nil
}

// WgStatParse - parse stats from parsed response.
// Most of fileds have a text format, so we need to parse them.
func WgStatParse(resp *WGStatsIn) (*WgStatTimestamp, *WgStatTrafficMap, *WgStatLastActivityMap, *WgStatEndpointMap, error) {
	ts, err := WgStatParseTimestamp(resp.Timestamp)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	trafficMap, err := WgStatParseTraffic(resp.Data.WgStatTrafficMapIn)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	lastActivityMap, err := WgStatParseLastActivity(resp.Data.WgStatLastseenMapIn)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	endpointsMap, err := WgStatParseEndpoints(resp.Data.WgStatEndpointMapIn)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	return ts, trafficMap, lastActivityMap, endpointsMap, nil
}
