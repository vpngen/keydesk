package storage

import (
	"os"
	"testing"
)

var db BrigadeStorage

func TestMain(m *testing.M) {
	mw := BrigadeTestMiddleware(&db, func(m *testing.M) int { return m.Run() })
	os.Exit(mw(m))
}

func TestSaveNotification(t *testing.T) {
	if err := db.CreateMessage("test"); err != nil {
		t.Errorf("save notification: %s", err)
	}
}

func TestPopNotifications(t *testing.T) {
	if err := db.CreateMessage("test"); err != nil {
		t.Errorf("save notification: %s", err)
	}

	messages, err := db.GetMessages()
	if err != nil {
		t.Errorf("get notifications: %s", err)
	}
	if len(messages) == 0 {
		t.Error("no notifications")
	}

	for _, message := range messages {
		t.Log(message)
	}
}
