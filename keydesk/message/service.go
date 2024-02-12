package message

import (
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/filter"
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

func paginate(messages []storage.Message, offset, limit int64) []storage.Message {
	if offset >= int64(len(messages)) {
		return nil
	}
	return messages[offset:min(offset+limit, int64(len(messages)))]
}

func (s Service) GetMessages(offset, limit int64, read *bool, priority *int64, priorityOp string) ([]storage.Message, int, error) {
	var result []storage.Message

	if err := s.transaction(func(brigade *storage.Brigade) error {
		result = brigade.Messages
		return nil
	}); err != nil {
		return nil, 0, err
	}

	var filters []filter.Func[storage.Message]

	if read != nil {
		filters = append(filters, isReadFilter(*read))
	}

	if priority != nil {
		filters = append(filters, priorityFilter(priorityOp, int(*priority)))
	}

	result = filter.Filter(result, filters...)

	return paginate(result, offset, limit), len(result), nil
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
	return filter.Filter(
		messages,
		ttlExpired(),
		firstN(10).IfOrTrue(noTTL()),
		notOlder(24*time.Hour*30).IfOrTrue(noTTL()),
		firstN(100),
	)
}
