package storage

import (
	"fmt"
	"time"
)

func (db *BrigadeStorage) GetMessages() ([]Message, error) {
	f, brigade, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("open to read: %w", err)
	}
	defer f.Close()

	if err := f.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return brigade.Messages, nil
}

func (db *BrigadeStorage) CreateMessage(text string) error {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	brigade.Messages = append(brigade.Messages, Message{
		Text:   text,
		IsRead: false,
		Time:   time.Now(),
	})

	if err := f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
