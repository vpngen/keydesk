package vapnapi

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type WgStatTimestamp struct {
	Timestamp int64
	Time      time.Time
}

type WgStatTraffic struct {
	WgPub   string
	WgRx    uint64
	WgTx    uint64
	IPSecRx uint64
	IPSecTx uint64
}

type WgStatTrafficMap map[string]*WgStatTraffic

type WgStatLastActivity struct {
	WgPub     string
	WgTime    time.Time
	IPSecTime time.Time
}

type WgStatLastActivityMap map[string]*WgStatLastActivity

type WgStatEndpoint struct {
	WgPub       string
	WgPrefix    netip.Prefix
	IPSecPrefix netip.Prefix
}

type WgStatEndpointMap map[string]*WgStatEndpoint

var (
	ErrInvalidStatFormat = errors.New("invalid stat")
)

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

func WgStatParseTraffic(traffic string) (WgStatTrafficMap, error) {
	var m = WgStatTrafficMap{}

	for _, line := range strings.Split(traffic, "\n") {
		if line == "" {
			continue
		}

		clmns := strings.Split(line, "\t")
		if len(clmns) < 3 {
			return nil, fmt.Errorf("traffic: %w", ErrInvalidStatFormat)
		}

		rx, err := strconv.ParseUint(clmns[1], 10, 64)
		if err != nil {
			continue
		}

		tx, err := strconv.ParseUint(clmns[2], 10, 64)
		if err != nil {
			continue
		}

		ep := &WgStatTraffic{
			WgPub: clmns[0],
			WgRx:  rx,
			WgTx:  tx,
		}

		if len(clmns) >= 5 {
			rx, err := strconv.ParseUint(clmns[3], 10, 64)
			if err != nil {
				continue
			}

			tx, err := strconv.ParseUint(clmns[4], 10, 64)
			if err != nil {
				continue
			}

			ep.IPSecRx = rx
			ep.IPSecTx = tx
		}

		m[clmns[0]] = ep
	}

	return m, nil
}

func WgStatParseLastActivity(lastSeen string) (WgStatLastActivityMap, error) {
	var m = WgStatLastActivityMap{}

	for _, line := range strings.Split(lastSeen, "\n") {
		if line == "" {
			continue
		}

		clmns := strings.Split(line, "\t")
		if len(clmns) < 2 {
			return nil, fmt.Errorf("last seen: %w", ErrInvalidStatFormat)
		}

		ep := &WgStatLastActivity{
			WgPub: clmns[0],
		}

		ts, err := strconv.ParseInt(clmns[1], 10, 64)
		if err == nil || ts != 0 {
			ep.WgTime = time.Unix(ts, 0).UTC()
		}

		if len(clmns) >= 3 {
			ts, err := strconv.ParseInt(clmns[2], 10, 64)
			if err == nil && ts != 0 {
				ep.IPSecTime = time.Unix(ts, 0).UTC()
			}
		}

		m[clmns[0]] = ep
	}

	return m, nil
}

func WgStatParseEndpoints(lastSeen string) (WgStatEndpointMap, error) {
	var m = WgStatEndpointMap{}

	for _, line := range strings.Split(lastSeen, "\n") {
		if line == "" {
			continue
		}

		clmns := strings.Split(line, "\t")
		if len(clmns) < 2 {
			return nil, fmt.Errorf("endpoints: %w", ErrInvalidStatFormat)
		}

		ep := &WgStatEndpoint{
			WgPub: clmns[0],
		}

		prefix, err := netip.ParsePrefix(clmns[1])
		if err == nil && !prefix.IsValid() {
			ep.WgPrefix = prefix
		}

		if len(clmns) >= 3 {
			prefix, err := netip.ParsePrefix(clmns[2])
			if err == nil && !prefix.IsValid() {
				ep.IPSecPrefix = prefix
			}
		}

		m[clmns[0]] = ep

	}

	return m, nil
}

func WgStatParse(resp *WGStats) (*WgStatTimestamp, WgStatTrafficMap, WgStatLastActivityMap, WgStatEndpointMap, error) {
	ts, err := WgStatParseTimestamp(resp.Timestamp)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	trafficMap, err := WgStatParseTraffic(resp.Traffic)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	lastActivityMap, err := WgStatParseLastActivity(resp.LastActivity)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	endpointsMap, err := WgStatParseEndpoints(resp.Endpoints)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse: %w", err)
	}

	return ts, trafficMap, lastActivityMap, endpointsMap, nil
}
