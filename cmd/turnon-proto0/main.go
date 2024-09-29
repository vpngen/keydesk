package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
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
	// ErrProto0AlreadyPresent - Proto0 already presents.
	ErrProto0AlreadyPresent = errors.New("proto0 already presents")
	// ErrProto0AlreadyAbsent - Proto0 already absent.
	ErrProto0AlreadyAbsent = errors.New("proto0 already absent")
)

func main() {
	replay, purge, brigadeID, dbDir, addr, domain, err := parseArgs()
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

	if err = Do(db, replay, purge, domain); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (bool, bool, string, string, netip.AddrPort, string, error) {
	var (
		id       string
		dbdir    string
		err      error
		addrPort netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, "", "", addrPort, "", fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	replay := flag.Bool("r", false, "Replay brigade")
	purge := flag.String("p", "", "Purge Protocol0 (need brigadeID)")
	domain := flag.String("dn", "", "Fake domain for OpenVPN over Cloak")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, "", "", addrPort, "", fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, "", "", addrPort, "", fmt.Errorf("addr: %w", err)
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

	return *replay, *purge == id, id, dbdir, addrPort, *domain, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, replay, purge bool, domain string) error {
	switch purge {
	case true:
		if err := removeProto0Support(db); err != nil {
			if errors.Is(err, ErrProto0AlreadyAbsent) {
				return nil
			}

			return fmt.Errorf("remove OVC: %w", err)
		}
	default:
		if err := addProto0Support(db, domain); err != nil {
			if errors.Is(err, ErrProto0AlreadyPresent) {
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

func addProto0Support(db *storage.BrigadeStorage, domain string) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	if data.Proto0FakeDomain != "" {
		fmt.Fprintf(os.Stderr, "Brigade %s already has Proto0\n", db.BrigadeID)

		return ErrProto0AlreadyPresent
	}

	proto0Conf := keydesk.GenEndpointProto0Creds(data.Proto0FakeDomain)

	if domain != "" {
		proto0Conf.Proto0FakeDomain = domain
	}

	data.CloakFakeDomain = proto0Conf.Proto0FakeDomain

	f.Commit(data)

	return nil
}

func removeProto0Support(db *storage.BrigadeStorage) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	if data.Proto0FakeDomain == "" {
		fmt.Fprintf(os.Stderr, "Brigade %s already hasn't Proto0\n", db.BrigadeID)

		return ErrProto0AlreadyAbsent
	}

	data.Proto0FakeDomain = ""

	f.Commit(data)

	return nil
}
