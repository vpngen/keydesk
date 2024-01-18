package storage

import (
	"github.com/SherClockHolmes/webpush-go"
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

func TestSaveSubscription(t *testing.T) {
	if err := db.SaveSubscription(webpush.Subscription{
		Endpoint: "test endpoint",
		Keys: webpush.Keys{
			P256dh: "test p256dh",
			Auth:   "test auth",
		},
	}); err != nil {
		t.Errorf("save subscription: %s", err)
	}

	sub, err := db.GetSubscription()
	if err != nil {
		t.Errorf("get subscription: %s", err)
	}

	if sub.Endpoint != "test endpoint" {
		t.Error("endpoint mismatch")
	}

	if sub.Keys.P256dh != "test p256dh" {
		t.Error("p256dh mismatch")
	}

	if sub.Keys.Auth != "test auth" {
		t.Error("auth mismatch")
	}
}
