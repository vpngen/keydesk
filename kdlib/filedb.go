package kdlib

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"

	"github.com/vpngen/keydesk/kdlib/lockedfile"
)

const (
	fileDbTempSuffix   = ".tmp"
	fileDbBackupSuffix = ".bak"
)

// FileDb - file pair as Db.
type FileDb struct {
	name   string
	r      *lockedfile.File
	w      *lockedfile.File
	unlock func()
}

// Commit - rename tmp to main and close all files.
func (f *FileDb) Commit() error {
	if err := f.w.Sync(); err != nil {
		return fmt.Errorf("sync temp: %w", err)
	}

	if err := os.Remove(f.name); err != nil {
		return fmt.Errorf("remove main: %w", err)
	}

	if err := os.Link(f.name+fileDbTempSuffix, f.name); err != nil {
		return fmt.Errorf("rename temp to name: %w", err)
	}

	if err := os.Remove(f.name + fileDbTempSuffix); err != nil {
		return fmt.Errorf("remove temp: %w", err)
	}

	return nil
}

// Close - rename tmp to main and close all files.
func (f *FileDb) Close() error {
	f.r.Close()
	f.w.Close()
	f.unlock()

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
			return fmt.Errorf("remove backup: %w", err)
		}
	}

	if err := os.Link(f.name, f.name+fileDbBackupSuffix); err != nil {
		return fmt.Errorf("link main to backup: %w", err)
	}

	return nil
}

// OpenFileDb - create file pair to edit.
func OpenFileDb(name, spinlock string, perm fs.FileMode) (*FileDb, error) {
	mu := lockedfile.MutexAt(spinlock)
	unlock, err := mu.Lock()
	if err != nil {
		return nil, fmt.Errorf("spinlock: %w", err)
	}

	w, err := lockedfile.OpenFile(name+fileDbTempSuffix, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return nil, fmt.Errorf("create temp: %w", err)
	}

	r, err := lockedfile.OpenFile(name, os.O_RDONLY|os.O_CREATE, perm)
	if err != nil {
		return nil, fmt.Errorf("read main: %w", err)
	}

	return &FileDb{
		name:   name,
		r:      r,
		w:      w,
		unlock: unlock,
	}, nil
}
