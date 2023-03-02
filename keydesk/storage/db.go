package storage

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"time"

	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/vapnapi"
)

// Filenames.
const (
	BrigadeFilename = "brigade.json"
	StatsFilename   = "stats.json"
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
	StatsFilename   string // i.e. /var/db/vgstats/<BrigadeID>/stat.json
	APIAddrPort     netip.AddrPort
	BrigadeStorageOpts
}

// pairFilesBrigadeStats - open and parsed data.
type pairFilesBrigadeStats struct {
	brigadeFile, statsFile *kdlib.FileDb
}

func (dt *pairFilesBrigadeStats) save(data *Brigade, stats *Stats) error {
	if err := dt.brigadeFile.Encoder(" ", " ").Encode(data); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.brigadeFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if err := dt.statsFile.Encoder(" ", " ").Encode(stats); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.statsFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (dt *pairFilesBrigadeStats) close() error {
	if err := dt.brigadeFile.Close(); err != nil {
		return fmt.Errorf("brigade: %w", err)
	}

	if err := dt.statsFile.Close(); err != nil {
		return fmt.Errorf("brigade: %w", err)
	}

	return nil
}

// SelfCheck - self check func.
func (db *BrigadeStorage) SelfCheck() error {
	if db.BrigadeFilename == "" ||
		db.StatsFilename == "" ||
		db.BrigadeID == "" ||
		db.MaxUsers == 0 ||
		db.ActivityPeriod == 0 ||
		db.MonthlyQuotaRemaining == 0 {
		return ErrWrongStorageConfiguration
	}

	return nil
}

func (db *BrigadeStorage) openStatsWithReading() (*kdlib.FileDb, *Stats, error) {
	f, err := kdlib.OpenFileDb(db.StatsFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	stats := &Stats{}

	err = f.Decoder().Decode(stats)
	if err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	if stats.BrigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return f, stats, nil
}

func (db *BrigadeStorage) openBrigadeWithReading() (*kdlib.FileDb, *Brigade, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	if err := f.Decoder().Decode(data); err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	// backup is read was succesfull.
	if err := f.Backup(); err != nil {
		return nil, nil, fmt.Errorf("backup: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return f, data, nil
}

func (db *BrigadeStorage) openWithReading() (*pairFilesBrigadeStats, *Brigade, *Stats, netip.AddrPort, error) {
	addr := netip.AddrPort{}

	fb, data, err := db.openBrigadeWithReading()
	if err != nil {
		return nil, nil, nil, addr, fmt.Errorf("brigade: %w", err)
	}

	fs, stats, err := db.openStatsWithReading()
	if err != nil {
		return nil, nil, nil, addr, fmt.Errorf("stats: %w", err)
	}

	calculatedAddrPort := vapnapi.CalcAPIAddrPort(data.EndpointIPv4)
	fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", calculatedAddrPort)

	switch {
	case db.APIAddrPort.Addr().IsValid() && db.APIAddrPort.Addr().IsUnspecified():
		addr = calculatedAddrPort
	default:
		addr = db.APIAddrPort
		if addr.IsValid() {
			fmt.Fprintf(os.Stderr, "API endpoint: %s\n", calculatedAddrPort)
		}
	}

	return &pairFilesBrigadeStats{
		brigadeFile: fb,
		statsFile:   fs,
	}, data, stats, addr, nil
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

func (db *BrigadeStorage) openStatsWithoutReading() (*kdlib.FileDb, *Stats, error) {
	f, err := kdlib.OpenFileDb(db.StatsFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	stats := &Stats{}

	err = f.Decoder().Decode(stats)
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

	return f, stats, nil
}

func (db *BrigadeStorage) openWithoutReading(brigadeID string) (*pairFilesBrigadeStats, *Brigade, *Stats, error) {
	if brigadeID != db.BrigadeID {
		return nil, nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	fb, data, err := db.openBrigadeWithoutReading()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("brigade: %w", err)
	}

	fs, stats, err := db.openStatsWithoutReading()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stat: %w", err)
	}

	data.Ver = BrigadeVersion
	stats.Ver = StatsVersion

	data.BrigadeID = brigadeID
	stats.BrigadeID = brigadeID

	ts := time.Now().UTC()
	data.CreatedAt = ts
	stats.BrigadeCreatedAt = ts

	return &pairFilesBrigadeStats{
		brigadeFile: fb,
		statsFile:   fs,
	}, data, stats, nil
}
