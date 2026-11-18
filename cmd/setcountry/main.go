package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
)

// ErrInvalidArgs - invalid arguments.
var ErrInvalidArgs = errors.New("invalid arguments")

// setcountry - store the brigade's country code in brigade.json. The keydesk
// puts it into the `country` JWT claim and the keydesk web picks the matching
// look ("ru" is the default look, "kg" the Kyrgyzstan one). Mirror of turnon-pro.
func main() {
	country, brigadeID, dbDir, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	fmt.Fprintf(os.Stderr, "Country: %s\n", country)

	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	if err := db.SetCountry(country); err != nil {
		log.Fatalf("Can't set country: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "Country is now: %s\n", db.Country())
}

func parseArgs() (string, string, string, error) {
	var (
		id    string
		dbdir string
		err   error
	)

	sysUser, err := user.Current()
	if err != nil {
		return "", "", "", fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	country := flag.String("country", "", "Country code: ru (default look) or kg")

	flag.Parse()

	if _, err := storage.NormalizeCountry(*country); err != nil {
		return "", "", "", fmt.Errorf("%w: %w", ErrInvalidArgs, err)
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return "", "", "", fmt.Errorf("dbdir dir: %w", err)
		}
	}

	switch *brigadeID {
	case "", sysUser.Username:
		id = sysUser.Username

		if *filedbDir == "" {
			dbdir = filepath.Join(storage.DefaultHomeDir, id)
		}
	default:
		id = *brigadeID

		cwd, err := os.Getwd()
		if err == nil {
			cwd, _ = filepath.Abs(cwd)
		}

		if *filedbDir == "" {
			dbdir = cwd
		}
	}

	return *country, id, dbdir, nil
}
