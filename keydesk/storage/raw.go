package storage

import (
	"fmt"
	"net/netip"

	"github.com/vpngen/keydesk/kdlib"
)

// RawOpenDbToModify - open FileDb to modify in raw tool.
type RawOpenDbToModify struct {
	f *kdlib.FileDb
}

// Close - close FileDb.
func (r *RawOpenDbToModify) Close() error {
	return r.f.Close()
}

// Commit - commit FileDb.
func (r *RawOpenDbToModify) Commit(data *Brigade) error {
	return commitBrigade(r.f, data)
}

// OpenDbToModify - open FileDb to modify.
func (db *BrigadeStorage) OpenDbToModify() (*RawOpenDbToModify, *Brigade, netip.AddrPort, error) {
	f, data, addr, err := db.openWithReading()
	if err != nil {
		return nil, nil, netip.AddrPort{}, fmt.Errorf("db: %w", err)
	}

	return &RawOpenDbToModify{f: f}, data, addr, nil
}
