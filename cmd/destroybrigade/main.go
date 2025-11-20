package main

import (
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
)

func parseArgs() (netip.AddrPort, string, string, bool, error) {
	var (
		addrPort netip.AddrPort
		dbdir    string
		id       string
	)

	sysUser, err := user.Current()
	if err != nil {
		return addrPort, "", "", false, fmt.Errorf("cannot define user: %w", err)
	}

	// is id only for debug?
	brigadeID := flag.String("id", "", "brigadier_id")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	force := flag.Bool("f", false, "force destroy VIP brigade")

	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return addrPort, "", "", false, fmt.Errorf("dbdir dir: %w", err)
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

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return addrPort, "", "", false, fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return addrPort, "", "", false, fmt.Errorf("id uuid: %s: %w", id, err)
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return addrPort, "", "", false, fmt.Errorf("api addr: %w", err)
		}
	}

	return addrPort, id, dbdir, *force, nil
}

func main() {
	addr, brigadeID, dbDir, force, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
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

	// just do it.
	if err := keydesk.DestroyBrigade(db, force); err != nil {
		log.Fatalf("Can't destroy brigade: %s\n", err)
	}
}
