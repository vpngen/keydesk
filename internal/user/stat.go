package user

import (
	"time"

	"github.com/google/uuid"
)

type (
	Activity struct {
		LastSeen time.Time `json:"last_seen"`

		TotalTraffic   uint64 `json:"total_traffic"`
		YearlyTraffic  uint64 `json:"yearly_traffic"`
		MonthlyTraffic uint64 `json:"monthly_traffic"`
		WeeklyTraffic  uint64 `json:"weekly_traffic"`
		DailyTraffic   uint64 `json:"daily_traffic"`
		PrevDayTraffic uint64 `json:"prev_day_traffic"`

		Updated time.Time `json:"updated"`
	}

	Activities map[uuid.UUID]Activity
)

func (s Service) GetLastConnections() (Activities, error) {
	users, err := s.db.ListUsers()
	if err != nil {
		return nil, err
	}
	res := make(map[uuid.UUID]Activity, len(users))
	for _, user := range users {
		res[user.UserID] = Activity{
			LastSeen: user.Quotas.LastActivity.Total,

			TotalTraffic:   user.Quotas.CountersTotal.Total.Rx + user.Quotas.CountersTotal.Total.Tx,
			YearlyTraffic:  user.Quotas.CountersTotal.Yearly.Rx + user.Quotas.CountersTotal.Yearly.Tx,
			MonthlyTraffic: user.Quotas.CountersTotal.Monthly.Rx + user.Quotas.CountersTotal.Monthly.Tx,
			WeeklyTraffic:  user.Quotas.CountersTotal.Weekly.Rx + user.Quotas.CountersTotal.Weekly.Tx,
			DailyTraffic:   user.Quotas.CountersTotal.Daily.Rx + user.Quotas.CountersTotal.Daily.Tx,
			PrevDayTraffic: user.Quotas.CountersTotal.PrevDay.Rx + user.Quotas.CountersTotal.PrevDay.Tx,

			Updated: user.Quotas.LastActivity.Update,
		}
	}
	return res, nil
}
