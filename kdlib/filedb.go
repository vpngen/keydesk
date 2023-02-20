package kdlib

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vpngen/keydesk/kdlib/lockedfile"
)

const (
	fileDbTempSuffix   = ".tmp"
	fileDbBackupSuffix = ".bak"
	fileDbperm         = 0640
)

// FileDb - file pair as Db.
type FileDb struct {
	name string
	r    *lockedfile.File
	w    *lockedfile.File
}

// Commit - rename tmp to main and close all files.
func (f *FileDb) Commit() error {
	if err := f.w.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	if err := os.Remove(f.name); err != nil {
		return fmt.Errorf("remove old: %w", err)
	}

	if err := os.Link(f.name+fileDbTempSuffix, f.name); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	if err := os.Remove(f.name + fileDbTempSuffix); err != nil {
		return fmt.Errorf("remove temp: %w", err)
	}

	return nil
}

// Close - rename tmp to main and close all files.
func (f *FileDb) Close() error {
	f.w.Close()
	f.r.Close()

	return nil
}

// Decoder - get json.Decoder (main file).
func (f *FileDb) Decoder() *json.Decoder {
	return json.NewDecoder(f.r)
}

// Encoder - get json.Encoder (tmp file).
func (f *FileDb) Encoder(prefix, indent string) *json.Encoder {
	enc := json.NewEncoder(f.w)

	enc.SetIndent(prefix, indent)

	return enc
}

// Backup - make a main file backup, probably after succesfull reading.
func (f *FileDb) Backup() error {
	if _, err := os.Stat(f.name + fileDbBackupSuffix); !os.IsNotExist(err) {
		if err := os.Remove(f.name + fileDbBackupSuffix); err != nil {
			return fmt.Errorf("remove: %w", err)
		}
	}

	if err := os.Link(f.name, f.name+fileDbBackupSuffix); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	return nil
}

// OpenFileDb - open file pair to edit.
func OpenFileDb(name string) (*FileDb, error) {
	w, err := lockedfile.OpenFile(name+fileDbTempSuffix, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileDbperm)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	r, err := lockedfile.OpenFile(name, os.O_RDWR|os.O_CREATE, fileDbperm)
	if err != nil {
		return nil, fmt.Errorf("edit: %w", err)
	}

	return &FileDb{
		name: name,
		r:    r,
		w:    w,
	}, nil
}
