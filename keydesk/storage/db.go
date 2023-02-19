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
	// ErrWrongStorageConfiguration - somthing empty in db config.
	ErrWrongStorageConfiguration = errors.New("wrong db config")
)

// BrigadeStorageOpts - opts.
type BrigadeStorageOpts struct {
	MaxUsers              int
	MonthlyQuotaRemaining int
	ActivityPeriod        time.Duration
}

// BrigadeStorage - brigade file storage.
type BrigadeStorage struct {
	BrigadeID       string
	BrigadeFilename string // i.e. /home/<BrigadeID>/brigade.json
	StatFilename    string // i.e. /var/db/vgstat/<BrigadeID>-stat.json
	APIAddrPort     netip.AddrPort
	BrigadeStorageOpts
}

// pairFilesBrigadeStat - open and parsed data.
type pairFilesBrigadeStat struct {
	brigadeFile, statFile *kdlib.FileDb
}

func (dt *pairFilesBrigadeStat) save(data *Brigade, stat *Stat) error {
	if err := dt.brigadeFile.Encoder(" ", " ").Encode(data); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.brigadeFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if err := dt.statFile.Encoder(" ", " ").Encode(stat); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.statFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (dt *pairFilesBrigadeStat) close() error {
	if err := dt.brigadeFile.Close(); err != nil {
		return fmt.Errorf("brigade: %w", err)
	}

	if err := dt.statFile.Close(); err != nil {
		return fmt.Errorf("brigade: %w", err)
	}

	return nil
}

// SelfCheck - self check func.
func (db *BrigadeStorage) SelfCheck() error {
	if db.BrigadeFilename == "" ||
		db.StatFilename == "" ||
		db.BrigadeID == "" ||
		db.MaxUsers == 0 ||
		db.ActivityPeriod == 0 ||
		db.MonthlyQuotaRemaining == 0 {
		return ErrWrongStorageConfiguration
	}

	return nil
}

func (db *BrigadeStorage) openStatWithReading() (*kdlib.FileDb, *Stat, error) {
	f, err := kdlib.OpenFileDb(db.StatFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	data := &Stat{}

	err = f.Decoder().Decode(data)
	if err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return f, data, nil
}

func (db *BrigadeStorage) openBrigadeWithReading() (*kdlib.FileDb, *Brigade, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	err = f.Decoder().Decode(data)
	if err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return f, data, nil
}

func (db *BrigadeStorage) openWithReading() (*pairFilesBrigadeStat, *Brigade, *Stat, netip.AddrPort, error) {
	addr := netip.AddrPort{}

	fb, data, err := db.openBrigadeWithReading()
	if err != nil {
		return nil, nil, nil, addr, fmt.Errorf("brigade: %w", err)
	}

	fs, stat, err := db.openStatWithReading()
	if err != nil {
		return nil, nil, nil, addr, fmt.Errorf("stat: %w", err)
	}

	addr = db.APIAddrPort
	if addr.Addr().IsValid() && addr.Addr().IsUnspecified() {
		addr = epapi.CalcAPIAddrPort(data.EndpointIPv4)
	}

	return &pairFilesBrigadeStat{
		brigadeFile: fb,
		statFile:    fs,
	}, data, stat, addr, nil
}

func (db *BrigadeStorage) openBrigadeWithoutReading() (*kdlib.FileDb, *Brigade, error) {
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

func (db *BrigadeStorage) openStatWithoutReading() (*kdlib.FileDb, *Stat, error) {
	f, err := kdlib.OpenFileDb(db.StatFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	stat := &Stat{}

	err = f.Decoder().Decode(stat)
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

	return f, stat, nil
}

func (db *BrigadeStorage) openWithoutReading(brigadeID string) (*pairFilesBrigadeStat, *Brigade, *Stat, error) {
	if brigadeID != db.BrigadeID {
		return nil, nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	fb, data, err := db.openBrigadeWithoutReading()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("brigade: %w", err)
	}

	fs, stat, err := db.openStatWithoutReading()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stat: %w", err)
	}

	data.BrigadeID = brigadeID
	stat.BrigadeID = brigadeID

	ts := time.Now()
	data.CreatedAt = ts
	stat.BrigadeCreatedAt = ts

	return &pairFilesBrigadeStat{
		brigadeFile: fb,
		statFile:    fs,
	}, data, stat, nil
}
