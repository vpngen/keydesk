package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"time"

	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/kdlib/lockedfile"
	"github.com/vpngen/keydesk/vapnapi"
)

// Filenames.
const (
	BrigadeFilename         = "brigade.json"
	StatsFilename           = "stats.json"
	KeydeskCountersFilename = "counters.json"
	QuotasFilename          = "quotas.json"
	FileDbMode              = 0644
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
	BrigadeID        string
	BrigadeFilename  string // i.e. /home/<BrigadeID>/brigade.json
	CountersFilename string // i.e. /home/<BrigadeID>/counter.json
	QuotasFilename   string // i.e. /var/lib/vgquotas/<BrigadeID>/quota.json
	APIAddrPort      netip.AddrPort
	BrigadeStorageOpts
}

// pairFilesBrigadeStats - open and parsed data.
type pairFilesBrigadeStats struct {
	brigadeFile, keydeskCountersFile *kdlib.FileDb
}

func (dt *pairFilesBrigadeStats) Save(data *Brigade, counters *KeydeskCounters) error {
	if err := dt.brigadeFile.Encoder(" ", " ").Encode(data); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.brigadeFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if err := dt.keydeskCountersFile.Encoder(" ", " ").Encode(counters); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.keydeskCountersFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (dt *pairFilesBrigadeStats) SaveCounters(counters *KeydeskCounters) error {
	if err := dt.keydeskCountersFile.Encoder(" ", " ").Encode(counters); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := dt.keydeskCountersFile.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (dt *pairFilesBrigadeStats) close() error {
	if err := dt.brigadeFile.Close(); err != nil {
		return fmt.Errorf("brigade: %w", err)
	}

	if err := dt.keydeskCountersFile.Close(); err != nil {
		return fmt.Errorf("counters: %w", err)
	}

	return nil
}

// SelfCheck - self check func.
func (db *BrigadeStorage) SelfCheck() error {
	if db.BrigadeFilename == "" ||
		db.CountersFilename == "" ||
		db.QuotasFilename == "" ||
		db.BrigadeID == "" ||
		db.MaxUsers == 0 ||
		db.ActivityPeriod == 0 ||
		db.MonthlyQuotaRemaining == 0 {
		return ErrWrongStorageConfiguration
	}

	return nil
}

func (db *BrigadeStorage) openCountersWithReading() (*kdlib.FileDb, *KeydeskCounters, error) {
	f, err := kdlib.OpenFileDb(db.QuotasFilename, FileDbMode)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	counters := &KeydeskCounters{}

	err = f.Decoder().Decode(counters)
	if err != nil {
		f.Close()

		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	if counters.BrigadeID != db.BrigadeID {
		return nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return f, counters, nil
}

func (db *BrigadeStorage) readQuotas() (*UsersQuotas, error) {
	f, err := lockedfile.OpenFile(db.QuotasFilename, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	defer f.Close()

	quotas := &UsersQuotas{}

	err = json.NewDecoder(f).Decode(quotas)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("decode: %w", err)
	}

	if quotas.BrigadeID != db.BrigadeID {
		return nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	return quotas, nil
}

func (db *BrigadeStorage) openBrigadeWithReading() (*kdlib.FileDb, *Brigade, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename, FileDbMode)
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

func (db *BrigadeStorage) openWithReading() (*pairFilesBrigadeStats, *Brigade, *KeydeskCounters, *UsersQuotas, netip.AddrPort, error) {
	addr := netip.AddrPort{}

	fb, data, err := db.openBrigadeWithReading()
	if err != nil {
		return nil, nil, nil, nil, addr, fmt.Errorf("brigade: %w", err)
	}

	fc, counters, err := db.openCountersWithReading()
	if err != nil {
		return nil, nil, nil, nil, addr, fmt.Errorf("stats: %w", err)
	}

	quotas, err := db.readQuotas()
	if err != nil {
		return nil, nil, nil, nil, addr, fmt.Errorf("quotas: %w", err)
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
		brigadeFile:         fb,
		keydeskCountersFile: fc,
	}, data, counters, quotas, addr, nil
}

func (db *BrigadeStorage) openBrigadeWithoutReading() (*kdlib.FileDb, *Brigade, error) {
	f, err := kdlib.OpenFileDb(db.BrigadeFilename, FileDbMode)
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

func (db *BrigadeStorage) openCounterWithoutReading() (*kdlib.FileDb, *KeydeskCounters, error) {
	f, err := kdlib.OpenFileDb(db.QuotasFilename, FileDbMode)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}

	counters := &KeydeskCounters{}

	err = f.Decoder().Decode(counters)
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

	return f, counters, nil
}

func (db *BrigadeStorage) openWithoutReading(brigadeID string) (*pairFilesBrigadeStats, *Brigade, *KeydeskCounters, error) {
	if brigadeID != db.BrigadeID {
		return nil, nil, nil, fmt.Errorf("check: %w", ErrUnknownBrigade)
	}

	fb, data, err := db.openBrigadeWithoutReading()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("brigade: %w", err)
	}

	fs, counters, err := db.openCounterWithoutReading()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stats: %w", err)
	}

	data.Ver = BrigadeVersion
	counters.Ver = KeydeskCountersVersion

	data.BrigadeID = brigadeID
	counters.BrigadeID = brigadeID

	ts := time.Now().UTC()
	data.CreatedAt = ts

	return &pairFilesBrigadeStats{
		brigadeFile:         fb,
		keydeskCountersFile: fs,
	}, data, counters, nil
}
