package kdlib

import "time"

func NextMonth(now time.Time) time.Time {
	year, month, _ := now.UTC().AddDate(0, 1, 0).Date()

	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}
