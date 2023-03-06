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
	WgPub string
	Rx    uint64
	Tx    uint64
}

type WgStatTrafficMap map[string]*WgStatTraffic

type WgStatLastActivity struct {
	WgPub string
	Time  time.Time
}

type WgStatLastActivityMap map[string]*WgStatLastActivity

type WgStatEndpoint struct {
	WgPub  string
	Prefix netip.Prefix
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
		if len(clmns) != 3 {
			return nil, fmt.Errorf("traffic: %w", ErrInvalidStatFormat)
		}

		rx, err := strconv.ParseUint(clmns[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("rx: %w", err)
		}

		tx, err := strconv.ParseUint(clmns[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("tx: %w", err)
		}

		m[clmns[0]] = &WgStatTraffic{
			WgPub: clmns[0],
			Rx:    rx,
			Tx:    tx,
		}
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
		if len(clmns) != 2 {
			return nil, fmt.Errorf("last seen: %w", ErrInvalidStatFormat)
		}

		ts, err := strconv.ParseInt(clmns[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("last seen: %w", err)
		}

		if ts == 0 {
			continue
		}

		m[clmns[0]] = &WgStatLastActivity{
			WgPub: clmns[0],
			Time:  time.Unix(ts, 0).UTC(),
		}
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
		if len(clmns) != 2 {
			return nil, fmt.Errorf("endpoints: %w", ErrInvalidStatFormat)
		}

		prefix, err := netip.ParsePrefix(clmns[1])
		if err != nil {
			continue
		}

		m[clmns[0]] = &WgStatEndpoint{
			WgPub:  clmns[0],
			Prefix: prefix,
		}
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
