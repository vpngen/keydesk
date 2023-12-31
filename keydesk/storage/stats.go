package storage

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/netip"
	"time"

	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/vpnapi"
)

var nullUnixTime = time.Unix(0, 0)

// GetStats - create brigade config.
func (db *BrigadeStorage) GetStats(rdata bool, statsFilename, statsSpinlock string, endpointsTTL time.Duration) error {
	data, err := db.getStatsQuota(rdata, endpointsTTL)
	if err != nil {
		return fmt.Errorf("quota: %w", err)
	}

	if err := db.putStatsStats(data, statsFilename, statsSpinlock); err != nil {
		return fmt.Errorf("stats: %w", err)
	}

	return nil
}

func lastActivityMark(now, lastActivity time.Time, points *LastActivityPoints) {
	defer func() {
		points.Update = now
	}()

	// !!! fix Unix zero time bug.
	if points.Total.Equal(nullUnixTime) {
		points.Total = time.Time{}
	}

	switch {
	case lastActivity.IsZero():
		if points.Total.IsZero() {
			return
		}

		lastActivity = points.Total
	case lastActivity.Equal(nullUnixTime):
		return
	default:
		points.Total = lastActivity
	}

	year, month, day := now.Date()
	lsYear, lsMonth, lsDay := lastActivity.Date()

	if lsYear == year && lsMonth == month && lsDay == day {
		points.Daily = lastActivity
		points.Weekly = lastActivity
		points.Monthly = lastActivity
		points.Yearly = lastActivity

		return
	}

	points.Daily = time.Time{}

	if lsYear != year {
		points.Weekly = time.Time{}
		points.Monthly = time.Time{}
		points.PrevMonthly = time.Time{}
		points.Yearly = time.Time{}

		return
	}

	points.Yearly = lastActivity

	switch {
	case lastActivity.Before(now.Add(-time.Hour * 24 * 7)):
		points.Weekly = time.Time{}
	case now.Weekday() < lastActivity.Weekday():
		points.Weekly = time.Time{}
	}

	if lsMonth != month {
		points.Monthly = time.Time{}

		_, prevMonth, _ := now.AddDate(0, -1, 0).Date()
		if lsMonth != prevMonth {
			points.PrevMonthly = time.Time{}
		}

		points.PrevMonthly = lastActivity

		return
	}

	points.Monthly = lastActivity
}

func incDateSwitchRelated(now time.Time, rx, tx uint64, counters *DateSummaryNetCounters) {
	defer func() {
		counters.Update = now
	}()

	counters.Total.Inc(rx, tx)

	if counters.Update.IsZero() {
		counters.Daily.Reset(rx, tx)
		counters.Weekly.Reset(rx, tx)
		counters.Monthly.Reset(rx, tx)
		counters.Yearly.Reset(rx, tx)

		return
	}

	prevYear, prevMonth, prevDay := counters.Update.Date()
	year, month, day := now.Date()

	if prevYear == year && prevMonth == month && prevDay == day {
		counters.Daily.Inc(rx, tx)
		counters.Weekly.Inc(rx, tx)
		counters.Monthly.Inc(rx, tx)
		counters.Yearly.Inc(rx, tx)

		return
	}

	if prevYear != year {
		counters.Yearly.Reset(0, 0)

		testYear, _, _ := counters.Update.AddDate(1, 0, 0).Date()
		if testYear != year {
			counters.Daily.Reset(0, 0)
			counters.Weekly.Reset(0, 0)
			counters.Monthly.Reset(0, 0)

			return
		}
	}

	counters.Yearly.Inc(rx, tx)

	switch {
	case counters.Update.Before(now.Add(-time.Hour * 24 * 7)):
		counters.Weekly.Reset(0, 0)
	case now.Weekday() < counters.Update.Weekday():
		counters.Weekly.Reset(0, 0)
	}

	counters.Weekly.Reset(rx, tx)

	if prevMonth != month {
		counters.Monthly.Reset(0, 0)

		testYear, testMonth, _ := counters.Update.AddDate(0, 1, 0).Date()
		if testYear != year || testMonth != month {
			counters.Daily.Reset(0, 0)

			return
		}
	}

	counters.Monthly.Inc(rx, tx)

	if prevDay != day {
		counters.Daily.Reset(0, 0)

		_, _, testDay := counters.Update.AddDate(0, 0, 1).Date()
		if testDay != day {
			return
		}
	}

	counters.Daily.Reset(rx, tx)
}

func randomData(data *Brigade, now time.Time) (*vpnapi.WgStatTimestamp, *vpnapi.WgStatTrafficMap, *vpnapi.WgStatLastActivityMap, *vpnapi.WgStatEndpointMap) {
	ts := &vpnapi.WgStatTimestamp{
		Time:      now,
		Timestamp: now.Unix(),
	}

	trafficMap := vpnapi.NewWgStatTrafficMap()
	lastSeenMap := vpnapi.NewWgStatLastActivityMap()
	endpointMap := vpnapi.NewWgStatEndpointMap()

	for _, user := range data.Users {
		id := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(user.WgPublicKey)

		switch rand.Int31n(20) {
		case 1:
			trafficMap.Wg[id] = &vpnapi.WgStatTraffic{
				Rx: uint64(rand.Int63n(1e4)),
				Tx: uint64(rand.Int63n(1e4)),
			}
			lastSeenMap.Wg[id] = now
			endpointMap.Wg[id] = netip.PrefixFrom(kdlib.RandomAddrIPv4(netip.PrefixFrom(netip.AddrFrom4([4]byte{0, 0, 0, 0}), 0)), 24)
		case 2:
			trafficMap.IPSec[id] = &vpnapi.WgStatTraffic{
				Rx: uint64(rand.Int63n(1e4)),
				Tx: uint64(rand.Int63n(1e4)),
			}
			lastSeenMap.IPSec[id] = now
			endpointMap.IPSec[id] = netip.PrefixFrom(kdlib.RandomAddrIPv4(netip.PrefixFrom(netip.AddrFrom4([4]byte{0, 0, 0, 0}), 0)), 24)
		}
	}

	return ts, trafficMap, lastSeenMap, endpointMap
}

func mergeStats(data *Brigade, wgStats *vpnapi.WGStatsIn, rdata bool, endpointsTTL, maxUserInactiveDuration time.Duration, monthlyQuotaRemaining int) error {
	var (
		totalTraffic TrafficCountersContainer
		throttledUsers,
		activeUsers,
		activeWgUsers,
		activeIPSecUsers,
		activeOvcUsers,
		activeOutlineUsers int
		trafficMap     *vpnapi.WgStatTrafficMap
		lastSeenMap    *vpnapi.WgStatLastActivityMap
		endpointMap    *vpnapi.WgStatEndpointMap
		statsTimestamp *vpnapi.WgStatTimestamp
		err            error
	)

	now := time.Now().UTC()

	switch rdata {
	case true:
		statsTimestamp, trafficMap, lastSeenMap, endpointMap = randomData(data, now)
	default:
		statsTimestamp, trafficMap, lastSeenMap, endpointMap, err = vpnapi.WgStatParse(wgStats)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}
	}

	for _, user := range data.Users {
		id := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(user.WgPublicKey)
		sum := RxTx{}

		if traffic, ok := trafficMap.Wg[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSWgCounters.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSWgCounters.Rx
			}

			if user.Quotas.OSWgCounters.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSWgCounters.Tx
			}

			user.Quotas.OSWgCounters.Rx = traffic.Rx
			user.Quotas.OSWgCounters.Tx = traffic.Tx

			sum.Inc(rx, tx)
			totalTraffic.TrafficWg.Inc(rx, tx)
			incDateSwitchRelated(now, rx, tx, &user.Quotas.CountersWg)
		}

		if traffic, ok := trafficMap.IPSec[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSIPSecCounters.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSIPSecCounters.Rx
			}

			if user.Quotas.OSIPSecCounters.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSIPSecCounters.Tx
			}

			user.Quotas.OSIPSecCounters.Rx = traffic.Rx
			user.Quotas.OSIPSecCounters.Tx = traffic.Tx

			sum.Inc(rx, tx)
			totalTraffic.TrafficIPSec.Inc(rx, tx)
			incDateSwitchRelated(now, rx, tx, &user.Quotas.CountersIPSec)
		}

		if traffic, ok := trafficMap.Ovc[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSOvcCounters.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSOvcCounters.Rx
			}

			if user.Quotas.OSOvcCounters.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSOvcCounters.Tx
			}

			user.Quotas.OSOvcCounters.Rx = traffic.Rx
			user.Quotas.OSOvcCounters.Tx = traffic.Tx

			sum.Inc(rx, tx)
			totalTraffic.TrafficOvc.Inc(rx, tx)
			incDateSwitchRelated(now, rx, tx, &user.Quotas.CountersOvc)
		}

		if traffic, ok := trafficMap.Outline[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSOutlineCounters.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSOutlineCounters.Rx
			}

			if user.Quotas.OSOutlineCounters.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSOutlineCounters.Tx
			}

			user.Quotas.OSOutlineCounters.Rx = traffic.Rx
			user.Quotas.OSOutlineCounters.Tx = traffic.Tx

			sum.Inc(rx, tx)
			totalTraffic.TrafficOutline.Inc(rx, tx)
			incDateSwitchRelated(now, rx, tx, &user.Quotas.CountersOutline)
		}

		totalTraffic.TrafficSummary.Inc(sum.Rx, sum.Tx)
		incDateSwitchRelated(now, sum.Rx, sum.Tx, &user.Quotas.CountersTotal)

		if user.Quotas.LimitMonthlyResetOn.Before(now) {
			// !!! reset monthly throttle ....
			user.Quotas.LimitMonthlyRemaining = uint64(monthlyQuotaRemaining)
			user.Quotas.LimitMonthlyResetOn = kdlib.NextMonthlyResetOn(now)
		}

		spentQuota := (sum.Rx + sum.Tx)
		switch {
		case user.Quotas.LimitMonthlyRemaining >= spentQuota:
			user.Quotas.LimitMonthlyRemaining -= spentQuota
		default:
			user.Quotas.LimitMonthlyRemaining = 0
		}

		// !!! fix Unix zero time bug.
		if user.Quotas.LastWgActivity.Total.Equal(nullUnixTime) {
			user.Quotas.LastWgActivity.Total = time.Time{}
		}
		lastActivityWg := lastSeenMap.Wg[id]
		lastActivityMark(now, lastActivityWg, &user.Quotas.LastWgActivity)

		// !!! fix Unix zero time bug.
		if user.Quotas.LastIPSecActivity.Total.Equal(nullUnixTime) {
			user.Quotas.LastIPSecActivity.Total = time.Time{}
		}
		lastActivityIPSec := lastSeenMap.IPSec[id]
		lastActivityMark(now, lastActivityIPSec, &user.Quotas.LastIPSecActivity)

		// !!! fix Unix zero time bug.
		if user.Quotas.LastOvcActivity.Total.Equal(nullUnixTime) {
			user.Quotas.LastOvcActivity.Total = time.Time{}
		}
		lastActivityOvc := lastSeenMap.Ovc[id]
		lastActivityMark(now, lastActivityOvc, &user.Quotas.LastOvcActivity)

		// !!! fix Unix zero time bug.
		if user.Quotas.LastOutlineActivity.Total.Equal(nullUnixTime) {
			user.Quotas.LastOutlineActivity.Total = time.Time{}
		}
		lastActivityOutline := lastSeenMap.Outline[id]
		lastActivityMark(now, lastActivityOutline, &user.Quotas.LastOutlineActivity)

		lastActivityTotal := user.Quotas.LastActivity.Total

		if lastActivityWg.After(lastActivityTotal) {
			lastActivityTotal = lastActivityWg
		}

		if lastActivityIPSec.After(lastActivityTotal) {
			lastActivityTotal = lastActivityIPSec
		}

		if lastActivityOvc.After(lastActivityTotal) {
			lastActivityTotal = lastActivityOvc
		}

		if lastActivityOutline.After(lastActivityTotal) {
			lastActivityTotal = lastActivityOutline
		}

		// !!! fix Unix zero time bug.
		if user.Quotas.LastActivity.Total.Equal(nullUnixTime) {
			user.Quotas.LastActivity.Total = time.Time{}
		}
		lastActivityMark(now, lastActivityTotal, &user.Quotas.LastActivity)

		if !user.Quotas.ThrottlingTill.IsZero() && user.Quotas.ThrottlingTill.After(now) {
			throttledUsers++
		}

		userInactiveEdge := now.Add(-maxUserInactiveDuration)

		if user.Quotas.LastActivity.Total.After(userInactiveEdge) {
			activeUsers++
		}

		if user.Quotas.LastWgActivity.Total.After(userInactiveEdge) {
			activeWgUsers++
		}

		if user.Quotas.LastIPSecActivity.Total.After(userInactiveEdge) {
			activeIPSecUsers++
		}

		if user.Quotas.LastOvcActivity.Total.After(userInactiveEdge) {
			activeOvcUsers++
		}

		if user.Quotas.LastOutlineActivity.Total.After(userInactiveEdge) {
			activeOutlineUsers++
		}
	}

	data.TotalUsersCount = len(data.Users)
	data.ThrottledUsersCount = throttledUsers
	data.ActiveUsersCount = activeUsers
	data.ActiveWgUsersCount = activeWgUsers
	data.ActiveIPSecUsersCount = activeIPSecUsers
	data.ActiveOvcUsersCount = activeOvcUsers
	data.ActiveOutlineUsersCount = activeOutlineUsers

	incDateSwitchRelated(now, totalTraffic.TrafficSummary.Rx, totalTraffic.TrafficSummary.Tx, &data.TotalTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficWg.Rx, totalTraffic.TrafficWg.Tx, &data.TotalWgTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficIPSec.Rx, totalTraffic.TrafficIPSec.Tx, &data.TotalIPSecTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficOvc.Rx, totalTraffic.TrafficOvc.Tx, &data.TotalOvcTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficOutline.Rx, totalTraffic.TrafficOutline.Tx, &data.TotalOutlineTraffic)

	if data.Endpoints == nil {
		data.Endpoints = UsersNetworks{}
	}

	for _, prefix := range endpointMap.Wg {
		if prefix.IsValid() {
			data.Endpoints[prefix.String()] = now
		}
	}

	for _, prefix := range endpointMap.IPSec {
		if prefix.IsValid() {
			data.Endpoints[prefix.String()] = now
		}
	}

	for _, prefix := range endpointMap.Ovc {
		if prefix.IsValid() {
			data.Endpoints[prefix.String()] = now
		}
	}

	lowLimit := now.Add(-endpointsTTL)
	for prefix, updated := range data.Endpoints {
		if updated.Before(lowLimit) {
			delete(data.Endpoints, prefix)
		}
	}

	data.CountersUpdateTime = statsTimestamp.Time

	if data.Ver < 6 {
		/*stats := &data.StatsCountersStack[len(data.StatsCountersStack)-1]
		stats.TotalTraffic = data.TotalTraffic.Monthly
		stats.TotalWgTraffic = data.TotalWgTraffic.Monthly
		stats.TotalIPSecTraffic = data.TotalIPSecTraffic.Monthly*/

		data.StatsCountersStack[0].ActiveWgUsersCount = data.StatsCountersStack[0].ActiveUsersCount

		for i := len(data.StatsCountersStack) - 1; i > 0; i-- {
			x := data.StatsCountersStack[i]
			y := data.StatsCountersStack[i-1]

			data.StatsCountersStack[i].TotalTraffic.Rx = x.NetCounters.TotalTraffic.Rx - y.NetCounters.TotalTraffic.Rx
			data.StatsCountersStack[i].TotalTraffic.Tx = x.NetCounters.TotalTraffic.Tx - y.NetCounters.TotalTraffic.Tx
			data.StatsCountersStack[i].TotalWgTraffic.Rx = x.NetCounters.TotalWgTraffic.Rx - y.NetCounters.TotalWgTraffic.Rx
			data.StatsCountersStack[i].TotalWgTraffic.Tx = x.NetCounters.TotalWgTraffic.Tx - y.NetCounters.TotalWgTraffic.Tx
			data.StatsCountersStack[i].TotalIPSecTraffic.Rx = x.NetCounters.TotalIPSecTraffic.Rx - y.NetCounters.TotalIPSecTraffic.Rx
			data.StatsCountersStack[i].TotalIPSecTraffic.Tx = x.NetCounters.TotalIPSecTraffic.Tx - y.NetCounters.TotalIPSecTraffic.Tx

			data.StatsCountersStack[i].ActiveWgUsersCount = data.StatsCountersStack[i].ActiveUsersCount
		}

		data.Ver = BrigadeVersion
	}

	if data.Ver < 7 || data.EndpointPort == 0 {
		data.EndpointPort = 51820
		data.Ver = BrigadeVersion
	}

	data.StatsCountersStack.Put(data.BrigadeCounters, totalTraffic)

	return nil
}

func (db *BrigadeStorage) getStatsQuota(rdata bool, endpointsTTL time.Duration) (*Brigade, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	// if we catch a slowdown problems we need organize queue
	wgStats, err := vpnapi.WgStat(data.BrigadeID, db.actualAddrPort, db.calculatedAddrPort, data.WgPublicKey)
	if err != nil {
		return nil, fmt.Errorf("wg stat: %w", err)
	}

	if wgStats != nil || rdata {
		if err := mergeStats(data, wgStats, rdata, endpointsTTL, db.MaxUserInctivityPeriod, db.MonthlyQuotaRemaining); err != nil {
			return nil, fmt.Errorf("merge stats: %w", err)
		}
	}

	err = commitBrigade(f, data)
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return data, nil
}

func (db *BrigadeStorage) putStatsStats(data *Brigade, statsFilename, statsSpinlock string) error {
	stats := &Stats{
		StatsCounters: StatsCounters{
			UsersCounters: data.UsersCounters,
			NetCounters: NetCounters{
				TotalTraffic:        data.TotalTraffic.Total,
				TotalWgTraffic:      data.TotalWgTraffic.Total,
				TotalIPSecTraffic:   data.TotalIPSecTraffic.Total,
				TotalOvcTraffic:     data.TotalOvcTraffic.Total,
				TotalOutlineTraffic: data.TotalOutlineTraffic.Total,
			},
			CountersUpdateTime: data.CountersUpdateTime,
		},
		BrigadeID:         data.BrigadeID,
		BrigadeCreatedAt:  data.CreatedAt,
		KeydeskFirstVisit: data.KeydeskFirstVisit,
		Endpoints:         data.Endpoints,
		UpdateTime:        time.Now().UTC(),
		Ver:               StatsVersion,
	}

	fs, err := openStats(statsFilename, statsSpinlock)
	if err != nil {
		return fmt.Errorf("open stats: %w", err)
	}

	defer fs.Close()

	if err = commitStats(fs, stats); err != nil {
		return fmt.Errorf("commit stats: %w", err)
	}

	return nil
}
