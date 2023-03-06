package storage

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/vpngen/keydesk/vapnapi"
)

// GetStats - create brigade config.
func (db *BrigadeStorage) GetStats(statsFilename string, endpointsTTL time.Duration) error {
	data, err := db.getStatsQuota(endpointsTTL)
	if err != nil {
		return fmt.Errorf("quota: %w", err)
	}

	if err := db.putStatsStats(data, statsFilename); err != nil {
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
		total             RxTx
		throttled, active int
	)

	statsTimestamp, trafficMap, lastSeenMap, endpointMap, err := vapnapi.WgStatParse(wgStats)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	inc := data.OSCountersUpdated != 0

	for _, user := range data.Users {
		id := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(user.WgPublicKey)

		if traffic, ok := trafficMap[id]; ok {
			rx := traffic.Rx
			tx := traffic.Tx

			if user.Quotas.OSCounters.Rx <= traffic.Rx {
				rx = traffic.Rx - user.Quotas.OSCounters.Rx
			}

			if user.Quotas.OSCounters.Tx <= traffic.Tx {
				tx = traffic.Tx - user.Quotas.OSCounters.Tx
			}

			user.Quotas.OSCounters.Rx = traffic.Rx
			user.Quotas.OSCounters.Tx = traffic.Tx

			if inc {
				total.Inc(rx, tx)

				incDateSwitchRelated(now, rx, tx, &user.Quotas.Counters)
				if user.Quotas.LimitMonthlyResetOn.Before(now) {
					user.Quotas.LimitMonthlyRemaining = uint64(monthlyQuotaRemaining)
					user.Quotas.LimitMonthlyResetOn = now.AddDate(0, 1, 0)
				}

				switch {
				case user.Quotas.LimitMonthlyRemaining >= (rx + tx):
					user.Quotas.LimitMonthlyRemaining -= (rx + tx)
				default:
					user.Quotas.LimitMonthlyRemaining = 0
				}
			}
		}

		if lastActivity, ok := lastSeenMap[id]; ok {
			lastActivityMark(now, lastActivity.Time, &user.Quotas.LastActivity)
		}

		if !user.Quotas.ThrottlingTill.IsZero() && user.Quotas.ThrottlingTill.After(now) {
			throttled++
		}

		if !user.Quotas.LastActivity.Monthly.IsZero() {
			active++
		}
	}

	data.ThrottledUserCount = throttled
	data.ActiveUsersCount = active

	if inc {
		incDateSwitchRelated(now, total.Rx, total.Tx, &data.TotalTraffic)
	}

	if data.Endpoints == nil {
		data.Endpoints = UsersNetworks{}
	}

	for _, prefix := range endpointMap {
		data.Endpoints[prefix.Prefix.String()] = now
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

	if err := mergeStats(data, wgStats, endpointsTTL, db.MonthlyQuotaRemaining); err != nil {
		return nil, fmt.Errorf("merge stats: %w", err)
	}

	err = CommitBrigade(f, data)
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return data, nil
}

func (db *BrigadeStorage) putStatsStats(data *Brigade, statsFilename string) error {
	stats := &Stats{
		BrigadeID:          data.BrigadeID,
		BrigadeCreatedAt:   data.CreatedAt,
		KeydeskLastVisit:   data.KeydeskLastVisit,
		UsersCount:         len(data.Users),
		ActiveUsersCount:   data.ActiveUsersCount,
		ThrottledUserCount: data.ThrottledUserCount,
		TotalTraffic:       data.TotalTraffic,
		Endpoints:          data.Endpoints,
		Ver:                StatsVersion,
	}

	fs, err := openStats(statsFilename)
	if err != nil {
		return fmt.Errorf("open stats: %w", err)
	}

	defer fs.Close()

	if err = CommitStats(fs, stats); err != nil {
		return fmt.Errorf("commit stats: %w", err)
	}

	return nil
}
