package push

import (
	"github.com/vpngen/keydesk/keydesk/storage"
)

type Service struct {
	db *storage.BrigadeStorage
}

func New(db *storage.BrigadeStorage) Service {
	return Service{
		db: db,
	}
}

func (s Service) SaveSubscription(sub storage.PushSubscription) error {
	return s.db.SaveSubscription(sub)
}

func (s Service) GetSubscription() (storage.PushSubscription, error) {
	return s.db.GetSubscription()
}
