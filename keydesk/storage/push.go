package storage

import (
	"fmt"
)

func (db *BrigadeStorage) SaveSubscription(sub PushSubscription) error {
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

var SubscriptionNotFound = fmt.Errorf("subscription not found")

func (db *BrigadeStorage) GetSubscription() (PushSubscription, error) {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return PushSubscription{}, fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	empty := PushSubscription{}
	if brigade.Subscription == empty {
		return PushSubscription{}, SubscriptionNotFound
	}

	sub := brigade.Subscription
	brigade.Subscription = empty

	if err := f.Commit(brigade); err != nil {
		return empty, fmt.Errorf("commit: %w", err)
	}

	return sub, nil
}
