package storage

import (
	"fmt"

	"github.com/vpngen/keydesk/vapnapi"
)

// GetStats - create brigade config.
func (db *BrigadeStorage) GetStats(statsFilename string) error {
	f, data, addr, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	// if we catch a slowdown problems we need organize queue
	_, err = vapnapi.WgStat(addr, data.WgPublicKey)
	if err != nil {
		return fmt.Errorf("wg stat: %w", err)
	}

	err = CommitBrigade(f, data)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	stats := &Stats{
		BrigadeID:        data.BrigadeID,
		BrigadeCreatedAt: data.CreatedAt,
		KeydeskLastVisit: data.KeydeskLastVisit,
		UsersCount:       len(data.Users),
		Ver:              StatsVersion,
	}

	fs, err := openStats(statsFilename)
	if err != nil {
		return fmt.Errorf("open stats: %w", err)
	}

	defer fs.Close()

	if err = CommitStats(fs, stats); err != nil {
		return fmt.Errorf("commit stats: %w", err)
	}

	return nil
}
