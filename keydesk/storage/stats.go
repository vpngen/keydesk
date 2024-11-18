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
	if now.Before(points.Update) {
		return
	}

	defer func() {
		points.Update = now
	}()

	// !!! fix Unix zero time bug.
	if points.Total.Equal(nullUnixTime) {
		points.Total = time.Time{}
	}

	switch {
	case lastActivity.IsZero(), lastActivity.Equal(nullUnixTime):
		if points.Total.IsZero() {
			return
		}

		lastActivity = points.Total
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

	lsWeekYear, lsWeek := lastActivity.ISOWeek()
	weekYear, week := now.ISOWeek()

	switch {
	case lsWeekYear == weekYear && lsWeek == week:
		points.Weekly = lastActivity
	case !points.Weekly.IsZero():
		lsWeekYear, lsWeek := points.Weekly.ISOWeek()
		if lsWeekYear != weekYear || lsWeek != week {
			points.Weekly = time.Time{}
		}
	}

	prevYear, prevMonth, _ := now.AddDate(0, -1, 0).Date()
	if !points.PrevMonthly.IsZero() {
		pmthYear, pmthMonth, _ := points.PrevMonthly.Date()
		if pmthYear != prevYear || pmthMonth != prevMonth {
			points.PrevMonthly = time.Time{}
		}
	}

	if !points.Monthly.IsZero() {
		mthYear, mthMonth, _ := points.Monthly.Date()
		if mthMonth != month {
			if mthYear == prevYear && mthMonth == prevMonth {
				points.PrevMonthly = points.Monthly
			}
		}
	}

	switch {
	case lsYear == year && lastActivity.After(points.Yearly):
		points.Yearly = lastActivity
	case !points.Yearly.IsZero():
		lsYear, _, _ := points.Yearly.Date()
		if lsYear != year {
			points.Yearly = time.Time{}
		}
	}

	switch {
	case lsYear == year && lsMonth == month && lastActivity.After(points.Monthly):
		points.Monthly = lastActivity
	case !points.Monthly.IsZero():
		lsYear, lsMonth, _ := points.Monthly.Date()
		if lsYear != year || lsMonth != month {
			points.Monthly = time.Time{}
		}
	}

	if !points.Daily.IsZero() {
		dYear, dMonth, dDay := points.Daily.Date()
		if dYear != year || dMonth != month || dDay != day {
			points.Daily = time.Time{}
		}
	}
}

func incDateSwitchRelated(now time.Time, rx, tx uint64, counters *DateSummaryNetCounters) {
	if now.Before(counters.Update) {
		return
	}

	defer func() {
		counters.Update = now
	}()

	counters.Total.Inc(rx, tx)

	if counters.Update.IsZero() {
		counters.PrevDay.Reset(0, 0)
		counters.Daily.Reset(0, 0)
		counters.Weekly.Reset(0, 0)
		counters.Monthly.Reset(0, 0)
		counters.Yearly.Reset(0, 0)

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

		if prevYear+1 != year {
			counters.Monthly.Reset(0, 0)
			counters.Weekly.Reset(0, 0)
			counters.Daily.Reset(0, 0)
			counters.PrevDay.Reset(0, 0)
		}
	}

	counters.Yearly.Inc(rx, tx)

	prevWeekYear, prevWeek := counters.Update.ISOWeek()
	weekYear, week := now.ISOWeek()

	if prevWeekYear != weekYear || prevWeek != week {
		counters.Weekly.Reset(0, 0)
	}

	counters.Weekly.Inc(rx, tx)

	if prevMonth != month {
		counters.Monthly.Reset(0, 0)

		testYear, testMonth, _ := counters.Update.AddDate(0, 1, 0).Date()
		if testYear != year || testMonth != month {
			counters.Daily.Reset(0, 0)
			counters.PrevDay.Reset(0, 0)
		}
	}

	counters.Monthly.Inc(rx, tx)

	if prevDay != day {
		counters.PrevDay.Reset(0, 0)

		testYear, testMonth, testDay := counters.Update.AddDate(0, 0, 1).Date()
		if testDay == day && testMonth == month && testYear == year {
			counters.PrevDay = counters.Daily
		}

		counters.Daily.Reset(0, 0)
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

func handleUserUsageStats(ustats *MarkedUsersCounters, quotas *Quota, limit uint64) {
	if quotas.CountersTotal.PrevDay.Rx > limit {
		ustats.TotalUsersCount++
	}

	if quotas.CountersWg.PrevDay.Rx > limit {
		ustats.WgUsersCount++
	}

	if quotas.CountersIPSec.PrevDay.Rx > limit {
		ustats.IPSecUsersCount++
	}

	if quotas.CountersOvc.PrevDay.Rx > limit {
		ustats.OvcUsersCount++
	}

	if quotas.CountersOutline.PrevDay.Rx > limit {
		ustats.OutlineUsersCount++
	}

	if quotas.CountersProto0.PrevDay.Rx > limit {
		ustats.Proto0UsersCount++
	}
}

func handleTrafficStat(
	id string,
	now time.Time,
	m map[string]*vpnapi.WgStatTraffic,
	osCounters *RxTx,
	sum *RxTx,
	total *RxTx,
	counters *DateSummaryNetCounters,
) {
	if traffic, ok := m[id]; ok {
		rx := traffic.Rx
		tx := traffic.Tx

		if osCounters.Rx <= traffic.Rx {
			rx = traffic.Rx - osCounters.Rx
		}

		if osCounters.Tx <= traffic.Tx {
			tx = traffic.Tx - osCounters.Tx
		}

		osCounters.Reset(traffic.Rx, traffic.Tx)

		sum.Inc(rx, tx)
		total.Inc(rx, tx)
		incDateSwitchRelated(now, rx, tx, counters)

		return
	}

	// reset OS counters.
	// osCounters.Reset(0, 0)
	// push zero traffic.
	incDateSwitchRelated(now, 0, 0, counters)
}

func handleLastActivity(
	id string,
	now time.Time,
	ausers *int,
	userInactiveEdge time.Time,
	lastSeenMap map[string]time.Time,
	endpointMap map[string]netip.Prefix,
	lastActivityPoints *LastActivityPoints,
	endpoints UsersNetworks,
	lastActivityTotal time.Time,
) time.Time {
	// !!! fix Unix zero time bug.
	if lastActivityPoints.Total.Equal(nullUnixTime) {
		lastActivityPoints.Total = time.Time{}
	}

	lastActivity := lastSeenMap[id]
	lastActivityMark(now, lastActivity, lastActivityPoints)

	if prefix, ok := endpointMap[id]; ok {
		if prefix.IsValid() {
			if endpoints[prefix.String()].Before(lastActivity) {
				endpoints[prefix.String()] = lastActivity
			}
		}
	}

	if lastActivityPoints.Total.After(userInactiveEdge) {
		*ausers++
	}

	if lastActivity.After(lastActivityTotal) {
		return lastActivity
	}

	return lastActivityTotal
}

func mergeStats(data *Brigade, wgStats *vpnapi.WGStatsIn, rdata bool, endpointsTTL, maxUserInactiveDuration time.Duration, monthlyQuotaRemaining int) error {
	var (
		totalTraffic TrafficCountersContainer

		users50gb   MarkedUsersCounters
		users100gb  MarkedUsersCounters
		users500gb  MarkedUsersCounters
		users1000gb MarkedUsersCounters

		blockedUsers,
		throttledUsers,
		activeUsers,
		activeWgUsers,
		activeIPSecUsers,
		activeOvcUsers,
		activeOlcUsers,
		activeOutlineUsers,
		activeProto0Users int

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

	if data.Endpoints == nil {
		data.Endpoints = UsersNetworks{}
	}

	for _, user := range data.Users {
		id := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(user.WgPublicKey)
		sum := RxTx{}

		handleTrafficStat(id, now, trafficMap.Wg, &user.Quotas.OSWgCounters, &sum, &totalTraffic.TrafficWg, &user.Quotas.CountersWg)
		handleTrafficStat(id, now, trafficMap.IPSec, &user.Quotas.OSIPSecCounters, &sum, &totalTraffic.TrafficIPSec, &user.Quotas.CountersIPSec)
		handleTrafficStat(id, now, trafficMap.Ovc, &user.Quotas.OSOvcCounters, &sum, &totalTraffic.TrafficOvc, &user.Quotas.CountersOvc)
		handleTrafficStat(id, now, trafficMap.Outline, &user.Quotas.OSOutlineCounters, &sum, &totalTraffic.TrafficOutline, &user.Quotas.CountersOutline)
		handleTrafficStat(id, now, trafficMap.Proto0, &user.Quotas.OSProto0Counters, &sum, &totalTraffic.TrafficProto0, &user.Quotas.CountersProto0)

		totalTraffic.TrafficSummary.Inc(sum.Rx, sum.Tx)
		incDateSwitchRelated(now, sum.Rx, sum.Tx, &user.Quotas.CountersTotal)

		handleUserUsageStats(&users50gb, &user.Quotas, 50*1024*1024*1024)
		handleUserUsageStats(&users100gb, &user.Quotas, 100*1024*1024*1024)
		handleUserUsageStats(&users500gb, &user.Quotas, 500*1024*1024*1024)
		handleUserUsageStats(&users1000gb, &user.Quotas, 1024*1024*1024*1024)

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

		lastActivityTotal := user.Quotas.LastActivity.Total
		userInactiveEdge := now.Add(-maxUserInactiveDuration)

		lastActivityTotal = handleLastActivity(id, now, &activeWgUsers, userInactiveEdge, lastSeenMap.Wg, endpointMap.Wg, &user.Quotas.LastWgActivity, data.Endpoints, lastActivityTotal)
		lastActivityTotal = handleLastActivity(id, now, &activeIPSecUsers, userInactiveEdge, lastSeenMap.IPSec, endpointMap.IPSec, &user.Quotas.LastIPSecActivity, data.Endpoints, lastActivityTotal)
		lastActivityTotal = handleLastActivity(id, now, &activeOvcUsers, userInactiveEdge, lastSeenMap.Ovc, endpointMap.Ovc, &user.Quotas.LastOvcActivity, data.Endpoints, lastActivityTotal)
		lastActivityTotal = handleLastActivity(id, now, &activeOlcUsers, userInactiveEdge, lastSeenMap.Olc, endpointMap.Olc, &user.Quotas.LastOlcActivity, data.Endpoints, lastActivityTotal)
		lastActivityTotal = handleLastActivity(id, now, &activeOutlineUsers, userInactiveEdge, lastSeenMap.Outline, endpointMap.Outline, &user.Quotas.LastOutlineActivity, data.Endpoints, lastActivityTotal)
		lastActivityTotal = handleLastActivity(id, now, &activeProto0Users, userInactiveEdge, lastSeenMap.Proto0, endpointMap.Proto0, &user.Quotas.LastProto0Activity, data.Endpoints, lastActivityTotal)

		// !!! fix Unix zero time bug.
		if user.Quotas.LastActivity.Total.Equal(nullUnixTime) {
			user.Quotas.LastActivity.Total = time.Time{}
		}
		lastActivityMark(now, lastActivityTotal, &user.Quotas.LastActivity)

		if !user.Quotas.ThrottlingTill.IsZero() && user.Quotas.ThrottlingTill.After(now) {
			throttledUsers++
		}

		if user.Quotas.LastActivity.Total.After(userInactiveEdge) {
			activeUsers++
		}

		if user.IsBlocked {
			blockedUsers++
		}
	}

	data.TotalUsersCount = len(data.Users)
	data.ThrottledUsersCount = throttledUsers
	data.BlockedUsersCount = blockedUsers
	data.ActiveUsersCount = activeUsers
	data.ActiveWgUsersCount = activeWgUsers
	data.ActiveIPSecUsersCount = activeIPSecUsers
	data.ActiveOvcUsersCount = activeOvcUsers
	data.ActiveOlcUsersCount = activeOlcUsers
	data.ActiveOutlineUsersCount = activeOutlineUsers
	data.ActiveProto0UsersCount = activeProto0Users

	incDateSwitchRelated(now, totalTraffic.TrafficSummary.Rx, totalTraffic.TrafficSummary.Tx, &data.TotalTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficWg.Rx, totalTraffic.TrafficWg.Tx, &data.TotalWgTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficIPSec.Rx, totalTraffic.TrafficIPSec.Tx, &data.TotalIPSecTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficOvc.Rx, totalTraffic.TrafficOvc.Tx, &data.TotalOvcTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficOutline.Rx, totalTraffic.TrafficOutline.Tx, &data.TotalOutlineTraffic)
	incDateSwitchRelated(now, totalTraffic.TrafficProto0.Rx, totalTraffic.TrafficProto0.Tx, &data.TotalProto0Traffic)

	data.Users50gb = users50gb
	data.Users100gb = users100gb
	data.Users500gb = users500gb
	data.Users1000gb = users1000gb

	lowLimit := now.Add(-endpointsTTL)
	for prefix, updated := range data.Endpoints {
		if updated.Before(lowLimit) {
			delete(data.Endpoints, prefix)
		}
	}

	data.CountersUpdateTime = statsTimestamp.Time

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
				TotalProto0Traffic:  data.TotalProto0Traffic.Total,
			},

			CountersUpdateTime: data.CountersUpdateTime,
		},

		Users50gb:   data.Users50gb,
		Users100gb:  data.Users100gb,
		Users500gb:  data.Users500gb,
		Users1000gb: data.Users1000gb,

		YesterdayTraffic: data.TotalTraffic.PrevDay,

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
