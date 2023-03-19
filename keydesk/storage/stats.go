package storage

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/vpngen/keydesk/vapnapi"
)

// GetStats - create brigade config.
func (db *BrigadeStorage) GetStats(statsFilename, statsSpinlock string, endpointsTTL time.Duration) error {
	data, err := db.getStatsQuota(endpointsTTL)
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

	switch {
	case lastActivity.IsZero():
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

func incDateSwitchRelated(now time.Time, rx, tx uint64, counters *NetCounters) {
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

func mergeStats(data *Brigade, wgStats *vapnapi.WGStats, endpointsTTL time.Duration, monthlyQuotaRemaining int) error {
	var (
		total, totalWg, totalIPSec               RxTx
		throttled, active, activeWg, activeIPSec int
	)

	statsTimestamp, trafficMap, lastSeenMap, endpointMap, err := vapnapi.WgStatParse(wgStats)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()

	for _, user := range data.Users {
		id := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(user.WgPublicKey)
		sum := RxTx{}

		if traffic, ok := trafficMap.Wg[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSCountersWg.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSCountersWg.Rx
			}

			if user.Quotas.OSCountersWg.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSCountersWg.Tx
			}

			user.Quotas.OSCountersWg.Rx = traffic.Rx
			user.Quotas.OSCountersWg.Tx = traffic.Tx

			sum.Inc(rx, tx)
			totalWg.Inc(rx, tx)
			incDateSwitchRelated(now, rx, tx, &user.Quotas.CountersWg)
		}

		if traffic, ok := trafficMap.IPSec[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSCounters.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSCountersIPSec.Rx
			}

			if user.Quotas.OSCounters.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSCountersIPSec.Tx
			}

			user.Quotas.OSCountersIPSec.Rx = traffic.Rx
			user.Quotas.OSCountersIPSec.Tx = traffic.Tx

			sum.Inc(rx, tx)
			totalIPSec.Inc(rx, tx)
			incDateSwitchRelated(now, rx, tx, &user.Quotas.CountersIPSec)
		}

		total.Inc(sum.Rx, sum.Tx)
		incDateSwitchRelated(now, sum.Rx, sum.Tx, &user.Quotas.CountersTotal)

		nextMonth := now.AddDate(0, 1, 0)
		if user.Quotas.LimitMonthlyResetOn.Before(now) {
			user.Quotas.LimitMonthlyRemaining = uint64(monthlyQuotaRemaining)
			// nextMonth := now.AddDate(0, 1, 0)
			// user.Quotas.LimitMonthlyResetOn = time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
		}
		// !!! force reset on next month.
		user.Quotas.LimitMonthlyResetOn = time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

		switch {
		case user.Quotas.LimitMonthlyRemaining >= (sum.Rx + sum.Tx):
			user.Quotas.LimitMonthlyRemaining -= (sum.Rx + sum.Tx)
		default:
			user.Quotas.LimitMonthlyRemaining = 0
		}

		// fix
		ts0 := time.Unix(0, 0)
		if user.Quotas.LastActivity.Total == ts0 {
			user.Quotas.LastActivity.Total = time.Time{}
		}
		if user.Quotas.LastActivityWg.Total == ts0 {
			user.Quotas.LastActivity.Total = time.Time{}
		}
		if user.Quotas.LastActivityIPSec.Total == ts0 {
			user.Quotas.LastActivity.Total = time.Time{}
		}

		lastActivityWg, ok := lastSeenMap.Wg[id]
		if ok {
			if !lastActivityWg.IsZero() {
				lastActivityMark(now, lastActivityWg, &user.Quotas.LastActivityWg)
			}
		}

		lastActivityIPSec, ok := lastSeenMap.IPSec[id]
		if ok {
			if !lastActivityIPSec.IsZero() {
				lastActivityMark(now, lastActivityIPSec, &user.Quotas.LastActivityIPSec)
			}
		}

		lastActivityTotal := lastActivityWg
		if lastActivityIPSec.After(lastActivityTotal) {
			lastActivityTotal = lastActivityIPSec
		}

		lastActivityMark(now, lastActivityTotal, &user.Quotas.LastActivity)

		if !user.Quotas.ThrottlingTill.IsZero() && user.Quotas.ThrottlingTill.After(now) {
			throttled++
		}

		if !user.Quotas.LastActivity.Monthly.IsZero() {
			active++
		}

		if !user.Quotas.LastActivityWg.Monthly.IsZero() {
			activeWg++
		}

		if !user.Quotas.LastActivityIPSec.Monthly.IsZero() {
			activeIPSec++
		}
	}

	data.ThrottledUserCount = throttled
	data.ActiveUsersCount = active
	data.ActiveUsersCountWg = activeWg
	data.ActiveUsersCountIPSec = activeIPSec

	incDateSwitchRelated(now, total.Rx, total.Tx, &data.TotalTraffic)
	incDateSwitchRelated(now, totalWg.Rx, totalWg.Tx, &data.TotalTrafficWg)
	incDateSwitchRelated(now, totalIPSec.Rx, totalIPSec.Tx, &data.TotalTrafficIPSec)

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

	lowLimit := now.Add(-endpointsTTL)
	for prefix, updated := range data.Endpoints {
		if updated.Before(lowLimit) {
			delete(data.Endpoints, prefix)
		}
	}

	data.OSCountersUpdated = statsTimestamp.Timestamp

	return nil
}

func (db *BrigadeStorage) getStatsQuota(endpointsTTL time.Duration) (*Brigade, error) {
	f, data, addr, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	// if we catch a slowdown problems we need organize queue
	wgStats, err := vapnapi.WgStat(addr, data.WgPublicKey)
	if err != nil {
		return nil, fmt.Errorf("wg stat: %w", err)
	}

	if wgStats != nil {
		if err := mergeStats(data, wgStats, endpointsTTL, db.MonthlyQuotaRemaining); err != nil {
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
		BrigadeID:          data.BrigadeID,
		BrigadeCreatedAt:   data.CreatedAt,
		KeydeskLastVisit:   data.KeydeskLastVisit,
		UsersCount:         len(data.Users),
		ActiveUsersCount:   data.ActiveUsersCount,
		ThrottledUserCount: data.ThrottledUserCount,
		TotalTraffic:       data.TotalTraffic,
		TotalTrafficWg:     data.TotalTrafficWg,
		TotalTrafficIPSec:  data.TotalTrafficIPSec,
		Endpoints:          data.Endpoints,
		Updated:            time.Now().UTC(),
		Ver:                StatsVersion,
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
