package storage

import "fmt"

func (db *BrigadeStorage) RunInTransaction(fn func(brigade *Brigade) error) error {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	if err = fn(brigade); err != nil {
		return fmt.Errorf("run in transaction: %w", err)
	}

	if err = f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
