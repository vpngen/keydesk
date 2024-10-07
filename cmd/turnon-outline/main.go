package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
)

var (
	// ErrInvalidArgs - invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")
	// ErrOutlineAlreadyPresent - IPSec already presents.
	ErrOutlineAlreadyPresent = errors.New("outline already presents")
	// ErrOutlineAlreadyAbsent - Outline already absent.
	ErrOutlineAlreadyAbsent = errors.New("outline already absent")
)

func main() {
	replay, purge, brigadeID, dbDir, addr, port, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	switch {
	case addr.IsValid() && !addr.Addr().IsUnspecified():
		fmt.Fprintf(os.Stderr, "Command address:port: %s\n", addr)
	case addr.IsValid():
		fmt.Fprintln(os.Stderr, "Command address:port is COMMON")
	default:
		fmt.Fprintln(os.Stderr, "Command address:port is for DEBUG")
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	if err = Do(db, replay, purge, port); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (bool, bool, string, string, netip.AddrPort, uint16, error) {
	var (
		id       string
		dbdir    string
		err      error
		addrPort netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, "", "", addrPort, 0, fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	replay := flag.Bool("r", false, "Replay brigade")
	purge := flag.String("p", "", "Purge IPSec (need brigadeID)")
	port := flag.Uint("op", 0, "Outline port, 0 is random")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, "", "", addrPort, 0, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, "", "", addrPort, 0, fmt.Errorf("addr: %w", err)
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

	return *replay, *purge == id, id, dbdir, addrPort, uint16(*port), nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, replay, purge bool, port uint16) error {
	switch purge {
	case true:
		if err := removeOutlineSupport(db); err != nil {
			if errors.Is(err, ErrOutlineAlreadyAbsent) {
				return nil
			}

			return fmt.Errorf("remove OVC: %w", err)
		}
	default:
		if err := addOutlineSupport(db, port); err != nil {
			if errors.Is(err, ErrOutlineAlreadyPresent) {
				return nil
			}

			return fmt.Errorf("apply OVC: %w", err)
		}
	}

	if replay {
		if err := db.ReplayBrigade(true, false, false); err != nil {
			return fmt.Errorf("replay brigade: %w", err)
		}
	}

	return nil
}

func addOutlineSupport(db *storage.BrigadeStorage, port uint16) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	if data.OutlinePort != 0 {
		fmt.Fprintf(os.Stderr, "Brigade %s already has Outline\n", db.BrigadeID)

		return ErrOutlineAlreadyPresent
	}

	if port == 0 {
		port = uint16(rand.Int31n(keydesk.HighOutlinePort-keydesk.LowOutlinePort) + keydesk.LowOutlinePort)
	}

	data.OutlinePort = port

	f.Commit(data)

	return nil
}

func removeOutlineSupport(db *storage.BrigadeStorage) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	if data.OutlinePort == 0 {
		fmt.Fprintf(os.Stderr, "Brigade %s already hasn't Outline\n", db.BrigadeID)

		return ErrOutlineAlreadyAbsent
	}

	data.OutlinePort = 0

	f.Commit(data)

	return nil
}
