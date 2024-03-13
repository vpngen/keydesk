package service

import (
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/filter"
	"time"
)

func noTTL() filter.Func[storage.Message] {
	return func(message storage.Message) bool {
		return message.TTL == 0
	}
}

func notOlder(d time.Duration) filter.Func[storage.Message] {
	t := time.Now().Add(-d)
	return func(message storage.Message) bool {
		return message.CreatedAt.After(t)
	}
}

func ttlAfterTime(t time.Time) filter.Func[storage.Message] {
	return func(message storage.Message) bool {
		return message.CreatedAt.Add(message.TTL).After(t)
	}
}

func ttlExpired() filter.Func[storage.Message] {
	now := time.Now()
	return ttlAfterTime(now).IfOrTrue(noTTL().Not())
}

func firstN(n int) filter.Func[storage.Message] {
	var i int
	return func(message storage.Message) bool {
		if i < n {
			i++
			return true
		}
		return false
	}
}

func isReadFilter(b bool) filter.Func[storage.Message] {
	return func(message storage.Message) bool {
		return message.IsRead == b
	}
}

func priorityFilter(op string, priority int) filter.Func[storage.Message] {
	return func(message storage.Message) bool {
		return filter.Ordered(op, priority)(message.Priority)
	}
}
