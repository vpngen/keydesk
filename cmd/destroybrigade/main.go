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
	"github.com/vpngen/keydesk/vapnapi"
)

func parseArgs() (netip.AddrPort, string, string, string, error) {
	var (
		addrPort        netip.AddrPort
		dbdir, statsdir string
		id              string
	)

	sysUser, err := user.Current()
	if err != nil {
		return addrPort, "", "", "", fmt.Errorf("cannot define user: %w", err)
	}

	// is id only for debug?
	brigadeID := flag.String("id", "", "brigadier_id")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	statsDir := flag.String("s", "", "Dir for statistic files (for test). Default: "+storage.DefaultStatsDir+"/<BrigadeID>")

	addr := flag.String("a", vapnapi.TemplatedAddrPort, "API endpoint address:port")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return addrPort, "", "", "", fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *statsDir != "" {
		statsdir, err = filepath.Abs(*statsDir)
		if err != nil {
			return addrPort, "", "", "", fmt.Errorf("statdir dir: %w", err)
		}
	}

	switch *brigadeID {
	case "", sysUser.Username:
		id = sysUser.Username

		if *filedbDir == "" {
			dbdir = filepath.Join(storage.DefaultHomeDir, id)
		}

		if *statsDir == "" {
			statsdir = filepath.Join(storage.DefaultStatsDir, id)
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

		if *statsDir == "" {
			statsdir = cwd
		}
	}

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return addrPort, "", "", "", fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return addrPort, "", "", "", fmt.Errorf("id uuid: %s: %w", id, err)
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return addrPort, "", "", "", fmt.Errorf("api addr: %w", err)
		}
	}

	return addrPort, id, dbdir, statsdir, nil
}

func main() {
	addr, brigadeID, dbDir, statsDir, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		StatsFilename:   filepath.Join(statsDir, storage.StatsFilename),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:              keydesk.MaxUsers,
			MonthlyQuotaRemaining: keydesk.MonthlyQuotaRemaining,
			ActivityPeriod:        keydesk.ActivityPeriod,
		},
	}
	if err := db.SelfCheck(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	// just do it.
	if err := keydesk.DestroyBrigade(db); err != nil {
		log.Fatalf("Can't destroy brigade: %s\n", err)
	}

}
