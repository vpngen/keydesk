package storage

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"time"

	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/vpnapi"
)

// Filenames.
const (
	BrigadeFilename         = "brigade.json"
	BrigadeSpinlockFilename = "brigade.lock"
	StatsFilename           = "stats.json"
	StatsSpinlockFilename   = "stats.lock"
	FileDbMode              = 0o644
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
	MaxUsers               int
	MonthlyQuotaRemaining  int
	MaxUserInctivityPeriod time.Duration
}

// BrigadeStorage - brigade file storage.
type BrigadeStorage struct {
	BrigadeID          string
	BrigadeFilename    string // i.e. /home/<BrigadeID>/brigade.json
	BrigadeSpinlock    string // i.e. /home/<BrigadeID>/brigade.lock
	APIAddrPort        netip.AddrPort
	calculatedAddrPort netip.AddrPort
	actualAddrPort     netip.AddrPort
	BrigadeStorageOpts
}

func commitBrigade(f *kdlib.FileDb, data *Brigade) error {
	if err := f.Encoder(" ", " ").Encode(data); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := f.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func commitStats(f *kdlib.FileDb, stats *Stats) error {
	if err := f.Encoder(" ", " ").Encode(stats); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := f.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// SelfCheck - self check func.
func (db *BrigadeStorage) SelfCheck() error {
	if db.BrigadeFilename == "" ||
		db.BrigadeSpinlock == "" ||
		db.BrigadeID == "" ||
		db.MaxUsers == 0 ||
		db.MaxUserInctivityPeriod == 0 ||
		db.MonthlyQuotaRemaining == 0 {
		return ErrWrongStorageConfiguration
	}

	return nil
}

// SelfCheckAndInit - self check and init func.
func (db *BrigadeStorage) SelfCheckAndInit() error {
	addr := netip.AddrPort{}

	if err := db.SelfCheck(); err != nil {
		return fmt.Errorf("self check: %w", err)
	}

	f, data, err := db.openBrigadeWithReading()
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer f.Close()

	db.calculatedAddrPort = vpnapi.CalcAPIAddrPort(data.EndpointIPv4)
	fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", db.calculatedAddrPort)

	switch {
	case db.APIAddrPort.Addr().IsValid() && db.APIAddrPort.Addr().IsUnspecified():
		db.actualAddrPort = db.calculatedAddrPort
	default:
		db.actualAddrPort = db.APIAddrPort
		if addr.IsValid() {
			fmt.Fprintf(os.Stderr, "API endpoint: %s\n", db.calculatedAddrPort)
		}
	}

	return nil
}

func (db *BrigadeStorage) CalculatedAPIAddress() (netip.Addr, bool) {
	return db.calculatedAddrPort.Addr(), db.APIAddrPort.Addr().IsValid() && db.APIAddrPort.Addr().IsUnspecified()
}

func (db *BrigadeStorage) openBrigadeWithReading() (*kdlib.FileDb, *Brigade, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename, db.BrigadeSpinlock, FileDbMode)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	data := &Brigade{}

	if err := f.Decoder().Decode(data); err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != db.BrigadeID {
		f.Close()

		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	// backup is read was succesfull.
	if err := f.Backup(); err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("backup: %w", err)
	}

	if data.Ver < 7 && data.EndpointPort == 0 {
		data.EndpointPort = 51820
	}

	return f, data, nil
}

func (db *BrigadeStorage) openWithReading() (*kdlib.FileDb, *Brigade, error) {
	f, data, err := db.openBrigadeWithReading()
	if err != nil {
		return nil, nil, fmt.Errorf("brigade: %w", err)
	}

	return f, data, nil
}

func (db *BrigadeStorage) openBrigadeWithoutReading() (*kdlib.FileDb, *Brigade, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename, db.BrigadeSpinlock, FileDbMode)
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

func (db *BrigadeStorage) openWithoutReading(brigadeID string) (*kdlib.FileDb, *Brigade, error) {
	if brigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	f, data, err := db.openBrigadeWithoutReading()
	if err != nil {
		return nil, nil, fmt.Errorf("brigade: %w", err)
	}

	ts := time.Now().UTC()
	data.Ver = BrigadeVersion
	data.BrigadeID = brigadeID
	data.CreatedAt = ts
	data.TotalTraffic = DateSummaryNetCounters{Ver: DateSummaryNetCountersVersion}
	data.TotalWgTraffic = DateSummaryNetCounters{Ver: DateSummaryNetCountersVersion}
	data.TotalIPSecTraffic = DateSummaryNetCounters{Ver: DateSummaryNetCountersVersion}

	return f, data, nil
}

func openStats(statsFilename, statsSpinlock string) (*kdlib.FileDb, error) {
	f, err := kdlib.OpenFileDb(statsFilename, statsSpinlock, FileDbMode)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	return f, nil
}
