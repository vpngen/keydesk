package message

import (
	"fmt"
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

func (s Service) transaction(fn func(brigade *storage.Brigade) error) error {
	f, brigade, err := s.db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	brigade.Messages = cleanupMessages(brigade.Messages)

	if err := fn(brigade); err != nil {
		return fmt.Errorf("run in transaction: %w", err)
	}

	if err := f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (s Service) GetMessages() ([]storage.Message, error) {
	var result []storage.Message

	if err := s.transaction(func(brigade *storage.Brigade) error {
		result = brigade.Messages
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (s Service) CreateMessage(text string, ttl time.Duration) error {
	return s.transaction(func(brigade *storage.Brigade) error {
		brigade.Messages = append(brigade.Messages, storage.Message{
			Text: text,
			Time: time.Now(),
			TTL:  ttl,
		})
		return nil
	})
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
