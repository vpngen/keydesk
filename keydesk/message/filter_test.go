package message

import (
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"testing"
	"time"
)

func stub() filterFunc[storage.Message] {
	return func(message storage.Message) bool {
		return true
	}

}
func Test_cleanupMessages(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		messages []storage.Message
		wantLen  int
	}{
		{
			"no messages",
			[]storage.Message{},
			0,
		},
		{
			"max 100",
			genMsg(101, time.Second, now),
			100,
		},
		{
			"last month",
			genMsg(100, time.Second, now),
			100,
		},
		{
			"last month",
			genMsg(100, 0, now.Add(-24*time.Hour*31)),
			0,
		},
		{
			"10 no ttl",
			genMsg(100, 0, now),
			10,
		},
		{
			"10 no ttl all",
			append(genMsg(100, 0, now), genMsg(100, time.Second, now)...),
			100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanupMessages(tt.messages); len(got) != tt.wantLen {
				t.Errorf("%s: got %d messages, want %d", tt.name, len(got), tt.wantLen)
			}
		})
	}
}

func genMsg(n int, ttl time.Duration, t time.Time) (messages []storage.Message) {
	for i := 0; i < n; i++ {
		messages = append(messages, storage.Message{
			Text: fmt.Sprintf("test-%d", i),
			TTL:  ttl,
			Time: t,
		})
	}
	return
}
