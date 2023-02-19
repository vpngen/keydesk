package storage

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/vpngen/keydesk/epapi"
	"github.com/vpngen/keydesk/kdlib"
)

// Filenames.
const (
	BrigadeFilename = "brigade.json"
	StatFilename    = "%s-stat.json"
)

var (
	// ErrUserLimit - maximun user num exeeded.
	ErrUserLimit = errors.New("num user limit exeeded")
	// ErrUserCollision - user name collision.
	ErrUserCollision = errors.New("username exists")
	// ErrBrigadierCollision - try to add more than one.
	ErrBrigadierCollision = errors.New("brigadier already exists")
	// ErrUnknownBrigade - brigade ID mismatch.
	ErrUnknownBrigade = errors.New("unknown brigade")
	// ErrBrigadeAlreadyExists - brigade file exists unexpectabily.
	ErrBrigadeAlreadyExists = errors.New("already exists")
)

// BrigadeStorage - brigade file storage.
type BrigadeStorage struct {
	BrigadeID       string
	BrigadeFilename string // i.e. /home/<BrigadeID>/brigade.json
	StatsFilename   string // i.e. /var/db/vgstat/<BrigadeID>-stat.json
	APIAddrPort     netip.AddrPort
}

func (db *BrigadeStorage) openWithReading() (*kdlib.FileDb, *Brigade, netip.AddrPort, error) {
	addr := netip.AddrPort{}

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, nil, addr, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		f.Close()

		return nil, nil, addr, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, nil, addr, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	addr = db.APIAddrPort
	if addr.Addr().IsValid() && addr.Addr().IsUnspecified() {
		addr = epapi.CalcAPIAddrPort(data.EndpointIPv4)
	}

	data.KeydeskLastVisit = time.Now()

	return f, data, addr, nil
}

func (db *BrigadeStorage) openWithoutReading(brigadeID string) (*kdlib.FileDb, *Brigade, error) {
	if brigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	switch err {
	case nil:
		f.Close()

		return nil, nil, fmt.Errorf("integrity: %w", ErrBrigadeAlreadyExists)
	case io.EOF:
		break
	default:
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	return f, data, nil
}

func (db *BrigadeStorage) save(f *kdlib.FileDb, data *Brigade) error {
	err := f.Encoder(" ", " ").Encode(data)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	err = f.Commit()
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
