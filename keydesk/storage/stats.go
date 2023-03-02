package storage

import "time"

func aggrStats(data *Brigade, stats *Stats, activityHours time.Duration) {
	var (
		active, throttled int
	)

	ts := time.Now().UTC()
	activityTime := ts.Add(-activityHours)

	stats.Updated = ts
	stats.UsersCount = len(data.Users)

	for _, user := range data.Users {
		if user.Quota.LastActivity.After(activityTime) {
			active++
		}

		if !user.Quota.ThrottlingTill.IsZero() {
			throttled++
		}
	}

	stats.ActiveUsersCount = active
	stats.ThrottledUserCount = throttled

	stats.TotalRx = data.CounterRX
	stats.TotalTx = data.CounterTX
}
