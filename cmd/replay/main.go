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

// ErrInvalidArgs - invalid arguments.
var ErrInvalidArgs = errors.New("invalid arguments")

func main() {
	fresh, bonly, uonly, erase, brigadeID, dbDir, addr, err := parseArgs()
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
			MaxUsers:              keydesk.MaxUsers,
			MonthlyQuotaRemaining: keydesk.MonthlyQuotaRemaining,
			ActivityPeriod:        keydesk.ActivityPeriod,
		},
	}
	if err := db.CheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	if err = Do(db, fresh, bonly, uonly, erase, addr); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (bool, bool, bool, bool, string, string, netip.AddrPort, error) {
	var (
		id       string
		dbdir    string
		err      error
		addrPort netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, false, false, "", "", addrPort, fmt.Errorf("cannot define user: %w", err)
	}

	bonly := flag.Bool("b", false, "brigades only, don't use with -u or -e flags")
	uonly := flag.Bool("u", false, "users only, don't use with -b or -r or -e flags")
	fresh := flag.Bool("r", false, "clean before (with deletion), don't use with -u or -e flags")
	erase := flag.Bool("e", false, "only delete brigades and users (without creation), don't use with any other mode flags")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")

	flag.Parse()

	if (*bonly && *uonly) || (*fresh && *uonly) || (*erase && *uonly) || (*erase && *bonly) || (*fresh && *erase) {
		return false, false, false, false, "", "", addrPort, ErrInvalidArgs
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, false, false, "", "", addrPort, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, false, false, "", "", addrPort, fmt.Errorf("addr: %w", err)
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

	return *fresh, *bonly, *uonly, *erase, id, dbdir, addrPort, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, fresh, bonly, uonly, erase bool, addr netip.AddrPort) error {
	if erase {
		if err := db.DestroyBrigade(); err != nil {
			return fmt.Errorf("destroy brigade: %w", err)
		}

		return nil
	}

	if err := db.ReplayBrigade(fresh, bonly, uonly); err != nil {
		return fmt.Errorf("replay brigade: %w", err)
	}

	return nil
}
