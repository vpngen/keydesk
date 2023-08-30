package vpnapi

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
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
	Wg    map[string]*WgStatTraffic
	IPSec map[string]*WgStatTraffic
	Ovc   map[string]*WgStatTraffic
}

// WgStatLastActivityMap - VPN stat last activity map, key is User wg_public_key.
// Dedicated map objects for wg and ipsec.
type WgStatLastActivityMap struct {
	Wg    map[string]time.Time
	IPSec map[string]time.Time
	Ovc   map[string]time.Time
}

// WgStatEndpointMap - VPN stat endpoint map, key is User wg_public_key.
// Dedicated map objects for wg and ipsec.
type WgStatEndpointMap struct {
	Wg    map[string]netip.Prefix
	IPSec map[string]netip.Prefix
	Ovc   map[string]netip.Prefix
}

// ErrInvalidStatFormat - invalid stat format.
var ErrInvalidStatFormat = errors.New("invalid stat")

// NewWgStatTrafficMap - create new WgStatTrafficMap.
func NewWgStatTrafficMap() *WgStatTrafficMap {
	return &WgStatTrafficMap{
		Wg:    make(map[string]*WgStatTraffic),
		IPSec: make(map[string]*WgStatTraffic),
		Ovc:   make(map[string]*WgStatTraffic),
	}
}

// NewWgStatLastActivityMap - create new WgStatLastActivityMap.
func NewWgStatLastActivityMap() *WgStatLastActivityMap {
	return &WgStatLastActivityMap{
		Wg:    make(map[string]time.Time),
		IPSec: make(map[string]time.Time),
		Ovc:   make(map[string]time.Time),
	}
}

// NewWgStatEndpointMap - create new WgStatEndpointMap.
func NewWgStatEndpointMap() *WgStatEndpointMap {
	return &WgStatEndpointMap{
		Wg:    make(map[string]netip.Prefix),
		IPSec: make(map[string]netip.Prefix),
		Ovc:   make(map[string]netip.Prefix),
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

func WgStatParseTrafficHandler(traffic string, traffic2 WgStatTrafficMap2) (*WgStatTrafficMap, error) {
	if traffic2 != nil {
		return WgStatParseTraffic2(traffic2)
	}

	return WgStatParseTraffic(traffic)
}

func WgStatParseTraffic2(traffic WgStatTrafficMap2) (*WgStatTrafficMap, error) {
	m := NewWgStatTrafficMap()

	for id, data := range traffic {
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
			case "wireguard":
				m.Wg[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			case "ipsec":
				m.IPSec[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			case "cloak-openvpn":
				m.Ovc[id] = &WgStatTraffic{
					Rx: rx,
					Tx: tx,
				}
			}
		}
	}

	return m, nil
}

// WgStatParseTraffic - parse traffic from text.
func WgStatParseTraffic(traffic string) (*WgStatTrafficMap, error) {
	m := NewWgStatTrafficMap()

	for _, line := range strings.Split(traffic, "\n") {
		if line == "" {
			continue
		}

		columns := strings.Split(line, "\t")
		if len(columns) < 3 {
			return nil, fmt.Errorf("traffic: %w", ErrInvalidStatFormat)
		}

		rx, err := strconv.ParseUint(columns[1], 10, 64)
		if err != nil {
			continue
		}

		tx, err := strconv.ParseUint(columns[2], 10, 64)
		if err != nil {
			continue
		}

		m.Wg[columns[0]] = &WgStatTraffic{
			Rx: rx,
			Tx: tx,
		}

		if len(columns) >= 5 {
			rx, err := strconv.ParseUint(columns[3], 10, 64)
			if err != nil {
				continue
			}

			tx, err := strconv.ParseUint(columns[4], 10, 64)
			if err != nil {
				continue
			}

			m.IPSec[columns[0]] = &WgStatTraffic{
				Rx: rx,
				Tx: tx,
			}
		}

		if len(columns) >= 7 {
			rx, err := strconv.ParseUint(columns[5], 10, 64)
			if err != nil {
				continue
			}

			tx, err := strconv.ParseUint(columns[6], 10, 64)
			if err != nil {
				continue
			}

			m.Ovc[columns[0]] = &WgStatTraffic{
				Rx: rx,
				Tx: tx,
			}
		}
	}

	return m, nil
}

func WgStatParseLastActivityHandler(lastSeen string, lastSeen2 WgStatLastseenMap2) (*WgStatLastActivityMap, error) {
	if lastSeen2 != nil {
		return WgStatParseLastActivity2(lastSeen2)
	}

	return WgStatParseLastActivity(lastSeen)
}

func WgStatParseLastActivity2(lastSeen WgStatLastseenMap2) (*WgStatLastActivityMap, error) {
	m := NewWgStatLastActivityMap()

	for id, data := range lastSeen {
		for vpnType, lastSeen := range data {
			ts, err := strconv.ParseInt(lastSeen.Timestamp, 10, 64)
			if err != nil {
				continue
			}

			switch vpnType {
			case "wireguard":
				m.Wg[id] = time.Unix(ts, 0).UTC()
			case "ipsec":
				m.IPSec[id] = time.Unix(ts, 0).UTC()
			case "cloak-openvpn":
				m.Ovc[id] = time.Unix(ts, 0).UTC()
			}
		}
	}

	return m, nil
}

// WgStatParseLastActivity - parse last activity time from text.
func WgStatParseLastActivity(lastSeen string) (*WgStatLastActivityMap, error) {
	m := NewWgStatLastActivityMap()

	for _, line := range strings.Split(lastSeen, "\n") {
		if line == "" {
			continue
		}

		columns := strings.Split(line, "\t")
		if len(columns) < 2 {
			return nil, fmt.Errorf("last seen: %w", ErrInvalidStatFormat)
		}

		ts, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			continue
		}

		if ts != 0 {
			m.Wg[columns[0]] = time.Unix(ts, 0).UTC()
		}

		if len(columns) >= 3 {
			ts, err := strconv.ParseInt(columns[2], 10, 64)
			if err != nil {
				continue
			}

			if ts != 0 {
				m.IPSec[columns[0]] = time.Unix(ts, 0).UTC()
			}
		}

		if len(columns) >= 4 {
			ts, err := strconv.ParseInt(columns[3], 10, 64)
			if err != nil {
				continue
			}

			if ts != 0 {
				m.Ovc[columns[0]] = time.Unix(ts, 0).UTC()
			}
		}
	}

	return m, nil
}

func WgStatParseEndpointsHandler(endpoints string, endpoints2 WgStatEndpointMap2) (*WgStatEndpointMap, error) {
	if endpoints2 != nil {
		return WgStatParseEndpoints2(endpoints2)
	}

	return WgStatParseEndpoints(endpoints)
}

func WgStatParseEndpoints2(endpoints WgStatEndpointMap2) (*WgStatEndpointMap, error) {
	m := NewWgStatEndpointMap()

	for id, data := range endpoints {
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
			case "wireguard":
				m.Wg[id] = prefix
			case "ipsec":
				m.IPSec[id] = prefix
			case "cloak-openvpn":
				m.Ovc[id] = prefix
			}
		}
	}

	return m, nil
}

// WgStatParseEndpoints - parse last seen endpoints from text.
func WgStatParseEndpoints(endpoints string) (*WgStatEndpointMap, error) {
	m := NewWgStatEndpointMap()

	for _, line := range strings.Split(endpoints, "\n") {
		if line == "" {
			continue
		}

		columns := strings.Split(line, "\t")
		if len(columns) < 2 {
			return nil, fmt.Errorf("endpoints: %w", ErrInvalidStatFormat)
		}

		prefix, err := netip.ParsePrefix(columns[1])
		if err != nil {
			continue
		}

		if prefix.IsValid() {
			m.Wg[columns[0]] = prefix
		}

		if len(columns) >= 3 {
			prefix, err := netip.ParsePrefix(columns[2])
			if err != nil {
				continue
			}

			if prefix.IsValid() {
				m.IPSec[columns[0]] = prefix
			}
		}

		if len(columns) >= 4 {
			prefix, err := netip.ParsePrefix(columns[2])
			if err != nil {
				continue
			}

			if prefix.IsValid() {
				m.Ovc[columns[0]] = prefix
			}
		}
	}

	return m, nil
}

// WgStatParse - parse stats from parsed response.
// Most of fileds have a text format, so we need to parse them.
func WgStatParse(resp *WGStats) (*WgStatTimestamp, *WgStatTrafficMap, *WgStatLastActivityMap, *WgStatEndpointMap, error) {
	ts, err := WgStatParseTimestamp(resp.Timestamp)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	trafficMap, err := WgStatParseTrafficHandler(resp.Traffic, resp.Data.WgStatTrafficMap2)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	lastActivityMap, err := WgStatParseLastActivityHandler(resp.LastActivity, resp.Data.WgStatLastseenMap2)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	endpointsMap, err := WgStatParseEndpointsHandler(resp.Endpoints, resp.Data.WgStatEndpointMap2)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	return ts, trafficMap, lastActivityMap, endpointsMap, nil
}
