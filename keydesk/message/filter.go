package message

import (
	"github.com/vpngen/keydesk/keydesk/storage"
	"time"
)

type filterFunc[T any] func(T) bool

func (f filterFunc[T]) and(f2 filterFunc[T]) filterFunc[T] {
	return func(v T) bool {
		return f(v) && f2(v)
	}
}

func (f filterFunc[T]) or(f2 filterFunc[T]) filterFunc[T] {
	return func(v T) bool {
		return f(v) || f2(v)
	}
}

func (f filterFunc[T]) not() filterFunc[T] {
	return func(t T) bool {
		return !f(t)
	}
}

func (f filterFunc[T]) ifOrTrue(f2 filterFunc[T]) filterFunc[T] {
	return func(v T) bool {
		if f2(v) {
			return f(v)
		}
		return true
	}
}

func (f filterFunc[T]) filter(values []T) (ret []T) {
	for _, v := range values {
		if f(v) {
			ret = append(ret, v)
		}
	}
	return
}

func filter[T any](values []T, filters ...filterFunc[T]) []T {
	for _, filter := range filters {
		values = filter.filter(values)
	}
	return values
}

func noTTL() filterFunc[storage.Message] {
	return func(message storage.Message) bool {
		return message.TTL == 0
	}
}

func notOlder(d time.Duration) filterFunc[storage.Message] {
	t := time.Now().Add(-d)
	return func(message storage.Message) bool {
		return message.Time.After(t)
	}
}

func ttlExpired() filterFunc[storage.Message] {
	now := time.Now()
	return noTTL().not().and(func(message storage.Message) bool {
		return message.Time.Add(message.TTL).After(now)
	}).or(noTTL())
}

func firstN(n int) filterFunc[storage.Message] {
	var i int
	return func(message storage.Message) bool {
		if i < n {
			i++
			return true
		}
		return false
	}
}
