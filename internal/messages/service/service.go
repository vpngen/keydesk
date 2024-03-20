package service

import (
	"errors"
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

func paginate(messages []storage.Message, offset, limit int) []storage.Message {
	if offset >= len(messages) {
		return nil
	}
	return messages[offset:min(offset+limit, len(messages))]
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

func messagePriorityLess(messages []storage.Message, asc bool) func(i, j int) bool {
	if asc {
		return func(i, j int) bool {
			return messages[i].Priority < messages[j].Priority
		}
	}
	return func(i, j int) bool {
		return messages[i].Priority > messages[j].Priority
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

	sortParams := make(map[string]bool)
	if sortTime != nil {
		sortParams = map[string]bool{"time": *sortTime == "desc"}
	}
	if sortPriority != nil {
		sortParams["priority"] = *sortPriority == "desc"
	}

	result, err := sortMessagesFactory(result, sortParams)
	if err != nil {
		return nil, 0, fmt.Errorf("sort messages: %w", err)
	}

	return paginate(result, int(offset), int(limit)), len(result), nil
}

//func (s Service) GetMessages2(
//	offset, limit int,
//	read *bool,
//	priority map[string]int,
//	sortParams map[string]bool,
//) ([]storage.Message, int, error) {
//	var result []storage.Message
//
//	if err := s.transaction(func(brigade *storage.Brigade) error {
//		result = brigade.Messages
//		return nil
//	}); err != nil {
//		return nil, 0, err
//	}
//
//	var filters []filter.Func[storage.Message]
//
//	if read != nil {
//		filters = append(filters, isReadFilter(*read))
//	}
//
//	for op, num := range priority {
//		filters = append(filters, priorityFilter(op, num))
//	}
//
//	result = filter.Filter(result, filters...)
//
//	result, err := sortMessagesFactory(result, sortParams)
//	if err != nil {
//		return nil, 0, fmt.Errorf("sort messages: %w", err)
//	}
//
//	return paginate(result, offset, limit), len(result), nil
//}

func sortFuncFactory(current, new func(i, j int) bool) func(i, j int) bool {
	if current == nil {
		return new
	}
	return func(i, j int) bool {
		return current(i, j) && new(i, j)
	}
}

func sortMessagesFactory(result []storage.Message, sortParams map[string]bool) ([]storage.Message, error) {
	var sortFn func(i, j int) bool
	for key, desc := range sortParams {
		switch key {
		case "time":
			sortFn = sortFuncFactory(sortFn, messageTimeLess(result, !desc))
		case "priority":
			sortFn = sortFuncFactory(sortFn, messagePriorityLess(result, !desc))
		default:
			return nil, fmt.Errorf("invalid sort key: %s", key)
		}
	}
	if sortFn != nil {
		sort.Slice(result, sortFn)
	}
	return result, nil
}

func (s Service) CreateMessage(text string, ttl time.Duration, priority int) (storage.Message, error) {
	var msg storage.Message
	if err := s.transaction(func(brigade *storage.Brigade) error {
		now := time.Now()
		msg = storage.Message{
			ID:        int(now.UnixNano()),
			Text:      text,
			Priority:  priority,
			CreatedAt: now,
			TTL:       ttl,
		}
		brigade.Messages = append(brigade.Messages, msg)
		return nil
	}); err != nil {
		return storage.Message{}, err
	}
	return msg, nil
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

var NotFound = errors.New("not found")

func (s Service) MarkAsRead(id int) error {
	return s.transaction(func(brigade *storage.Brigade) error {
		for i, message := range brigade.Messages {
			if message.ID == id {
				brigade.Messages[i].IsRead = true
				return nil
			}
		}
		return NotFound
	})
}
