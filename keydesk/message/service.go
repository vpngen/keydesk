package message

import (
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/filter"
	"sort"
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

	if err = fn(brigade); err != nil {
		return fmt.Errorf("run in transaction: %w", err)
	}

	if err = f.Commit(brigade); err != nil {
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

func messageTimeLess(messages []storage.Message, asc bool) func(i, j int) bool {
	if asc {
		return func(i, j int) bool {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
	}
	return func(i, j int) bool {
		return messages[i].CreatedAt.After(messages[j].CreatedAt)
	}
}

func (s Service) GetMessages(
	offset, limit int64,
	read *bool,
	priority *int64, priorityOp string,
	sortTime, sortPriority *string,
) ([]storage.Message, int, error) {
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

	result, err := sortMessages(result, sortTime, sortPriority)
	if err != nil {
		return nil, 0, fmt.Errorf("sort messages: %w", err)
	}

	return paginate(result, offset, limit), len(result), nil
}

func sortMessages(result []storage.Message, sortTime, sortPriority *string) ([]storage.Message, error) {
	var sortTimeFn func(i, j int) bool
	if sortTime != nil {
		switch *sortTime {
		case "asc":
			sortTimeFn = messageTimeLess(result, true)
		case "desc":
			sortTimeFn = messageTimeLess(result, false)
		default:
			return nil, fmt.Errorf("invalid sort time: %s", *sortTime)
		}
	}

	var sortPriorityFn func(i, j int) bool
	if sortPriority != nil {
		switch *sortPriority {
		case "asc":
			sortPriorityFn = func(i, j int) bool {
				return result[i].Priority < result[j].Priority
			}
		case "desc":
			sortPriorityFn = func(i, j int) bool {
				return result[i].Priority > result[j].Priority
			}
		default:
			return nil, fmt.Errorf("invalid sort priority: %s", *sortPriority)
		}
	}

	var sortFn func(i, j int) bool
	if sortTimeFn != nil {
		sortFn = sortTimeFn
	}
	if sortPriorityFn != nil {
		if sortFn == nil {
			sortFn = sortPriorityFn
		} else {
			sortFn = func(i, j int) bool {
				return sortPriorityFn(i, j) && sortTimeFn(i, j)
			}
		}
	}

	if sortFn != nil {
		sort.Slice(result, sortFn)
	}

	return result, nil
}

func (s Service) CreateMessage(text string, ttl time.Duration, priority int) error {
	return s.transaction(func(brigade *storage.Brigade) error {
		brigade.Messages = append(brigade.Messages, storage.Message{
			Text:      text,
			Priority:  priority,
			CreatedAt: time.Now(),
			TTL:       ttl,
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
