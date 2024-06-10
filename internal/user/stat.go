package user

import (
	"github.com/google/uuid"
	"time"
)

type (
	Activity struct {
		LastSeen time.Time `json:"last_seen"`
		Updated  time.Time `json:"updated"`
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
			Updated:  user.Quotas.LastActivity.Update,
		}
	}
	return res, nil
}
