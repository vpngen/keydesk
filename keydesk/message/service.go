package message

import (
	"github.com/vpngen/keydesk/keydesk/storage"
	"time"
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

func cleanupMessages(messages []storage.Message) []storage.Message {
	return filter(
		messages,
		ttlExpired(),
		firstN(10).ifOrTrue(noTTL()),
		notOlder(24*time.Hour*30).ifOrTrue(noTTL()),
		firstN(100),
	)
}
