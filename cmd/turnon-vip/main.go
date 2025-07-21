package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
)

// ErrInvalidArgs - invalid arguments.
var ErrInvalidArgs = errors.New("invalid arguments")

func main() {
	on, off, brigadeID, dbDir, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)

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

	if err = Do(db, on, off); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (bool, bool, string, string, error) {
	var (
		id    string
		dbdir string
		err   error
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, "", "", fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")

	flag.Parse()

	if flag.NArg() != 1 {
		return false, false, "", "", ErrInvalidArgs
	}

	arg := strings.ToLower(flag.Arg(0))
	if arg != "on" && arg != "off" {
		return false, false, "", "", fmt.Errorf("invalid argument: %s, expected 'on' or 'off'", arg)
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, "", "", fmt.Errorf("dbdir dir: %w", err)
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

	return arg == "on", arg == "off", id, dbdir, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, on, off bool) error {
	if on && off {
		return fmt.Errorf("cannot turn on and off at the same time")
	}

	if !on && !off {
		return fmt.Errorf("no action specified, use 'on' or 'off'")
	}

	if on {
		if err := db.SetVIP(true); err != nil {
			return fmt.Errorf("set VIP: %w", err)
		}
	}

	if off {
		if err := db.SetVIP(false); err != nil {
			return fmt.Errorf("unset VIP: %w", err)
		}
	}

	return nil
}
