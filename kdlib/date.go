package kdlib

import "time"

// NextMonthlyResetOn - returns the time of the next monthly reset LimitMonthlyResetOn value.
func NextMonthlyResetOn(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}
