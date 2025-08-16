package storage

import (
	"fmt"
	"sync/atomic"
)

func (db *BrigadeStorage) SetVIP(vip bool) error {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	switch vip {
	case true:
		atomic.StoreInt64(&brigade.VIP, 1)
	default:
		atomic.StoreInt64(&brigade.VIP, 0)
	}

	if err := f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (db *BrigadeStorage) IsVIP() bool {
	f, brigade, err := db.openWithReading()
	if err != nil {
		return false
	}
	defer f.Close()

	if brigade == nil {
		return false
	}

	vip := atomic.LoadInt64(&brigade.VIP)

	return vip > 0
}
