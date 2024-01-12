package message

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

func (s Service) GetMessages() ([]storage.Message, error) {
	return s.db.GetMessages()
}

func (s Service) CreateMessage(text string) error {
	return s.db.CreateMessage(text)
}
