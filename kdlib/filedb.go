package kdlib

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vpngen/keydesk/kdlib/lockedfile"
)

const (
	fileDbsuffix = ".tmp"
	fileDbperm   = 0640
)

// FileDb - file pair as Db.
type FileDb struct {
	name string
	r    *lockedfile.File
	w    *lockedfile.File
}

// Commit - rename tmp to main and close all files.
func (f *FileDb) Commit() error {
	err := f.w.Sync()
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	err = os.Rename(f.name+fileDbsuffix, f.name)
	if err != nil {
		return fmt.Errorf("rename: %w", err)
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

// OpenFileDb - open file pair to edit.
func OpenFileDb(name string) (*FileDb, error) {
	w, err := lockedfile.OpenFile(name+fileDbsuffix, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileDbperm)
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
