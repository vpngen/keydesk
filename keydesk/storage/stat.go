package storage

import "time"

func aggrStat(data *Brigade, stat *Stat, activityHours time.Duration) {
	var (
		active, throttled int
	)

	ts := time.Now().UTC()
	activityTime := ts.Add(-activityHours)

	stat.Updated = ts
	stat.UsersCount = len(data.Users)

	for _, user := range data.Users {
		if user.Quota.LastActivity.After(activityTime) {
			active++
		}

		if !user.Quota.ThrottlingTill.IsZero() {
			throttled++
		}
	}

	stat.ActiveUsersCount = active
	stat.ThrottledUserCount = throttled

	stat.TotalRx = data.CounterRX
	stat.TotalTx = data.CounterTX
}
