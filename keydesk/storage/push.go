package storage

import (
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
)

func (db *BrigadeStorage) SaveSubscription(sub webpush.Subscription) error {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	brigade.Subscription = sub

	if err := f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

var ErrSubscriptionNotFound = fmt.Errorf("subscription not found")

func (db *BrigadeStorage) GetSubscription() (webpush.Subscription, error) {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return webpush.Subscription{}, fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	empty := webpush.Subscription{}
	if brigade.Subscription == empty {
		return webpush.Subscription{}, ErrSubscriptionNotFound
	}

	sub := brigade.Subscription
	brigade.Subscription = empty

	if err := f.Commit(brigade); err != nil {
		return empty, fmt.Errorf("commit: %w", err)
	}

	return sub, nil
}
