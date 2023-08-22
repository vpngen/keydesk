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
	"github.com/vpngen/vpngine/naclkey"
)

var (
	// ErrInvalidArgs - invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")
	// ErrIPSecAlreadyPresent - IPSec already presents.
	ErrIPSecAlreadyPresent = errors.New("ipsec already presents")
	// ErrIPSecAlreadyAbsent - IPSec already absent.
	ErrIPSecAlreadyAbsent = errors.New("ipsec already absent")
)

func main() {
	replay, purge, brigadeID, etcDir, dbDir, addr, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
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

	if err = Do(db, replay, purge, &routerPublicKey, &shufflerPublicKey); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (bool, bool, string, string, string, netip.AddrPort, error) {
	var (
		id       string
		dbdir    string
		etcdir   string
		err      error
		addrPort netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, "", "", "", addrPort, fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	replay := flag.Bool("r", false, "Replay brigade")
	purge := flag.String("p", "", "Purge IPSec (need brigadeID)")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, "", "", "", addrPort, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return false, false, "", "", "", addrPort, fmt.Errorf("etcdir dir: %w", err)
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, "", "", "", addrPort, fmt.Errorf("addr: %w", err)
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

	return *replay, *purge == id, id, etcdir, dbdir, addrPort, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, replay, purge bool, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) error {
	switch purge {
	case true:
		if err := removeIPSecSupport(db, routerPublicKey, shufflerPublicKey); err != nil {
			if errors.Is(err, ErrIPSecAlreadyAbsent) {
				return nil
			}

			return fmt.Errorf("remove OVC: %w", err)
		}
	default:
		if err := addIPSecSupport(db, routerPublicKey, shufflerPublicKey); err != nil {
			if errors.Is(err, ErrIPSecAlreadyPresent) {
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

func addIPSecSupport(db *storage.BrigadeStorage, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	if data.IPSecPSK != "" && data.IPSecPSKRouterEnc != "" && data.IPSecPSKShufflerEnc != "" {
		fmt.Fprintf(os.Stderr, "Brigade %s already has IPSec\n", db.BrigadeID)

		return ErrIPSecAlreadyPresent
	}

	ipsecConf, err := keydesk.GenEndpointIPSecCreds(routerPublicKey, shufflerPublicKey)
	if err != nil {
		return fmt.Errorf("ipsec creds: %w", err)
	}

	data.IPSecPSK = ipsecConf.IPSecPSK
	data.IPSecPSKRouterEnc = ipsecConf.IPSecPSKRouterEnc
	data.IPSecPSKShufflerEnc = ipsecConf.IPSecPSKShufflerEnc

	f.Commit(data)

	return nil
}

func removeIPSecSupport(db *storage.BrigadeStorage, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	if data.IPSecPSK == "" && data.IPSecPSKRouterEnc == "" && data.IPSecPSKShufflerEnc == "" {
		fmt.Fprintf(os.Stderr, "Brigade %s already hasn't IPSec\n", db.BrigadeID)

		return ErrIPSecAlreadyAbsent
	}

	data.IPSecPSK = ""
	data.IPSecPSKRouterEnc = ""
	data.IPSecPSKShufflerEnc = ""

	f.Commit(data)

	return nil
}

func readPubKeys(path string) ([naclkey.NaclBoxKeyLength]byte, [naclkey.NaclBoxKeyLength]byte, error) {
	empty := [naclkey.NaclBoxKeyLength]byte{}

	routerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, keydesk.RouterPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("router key: %w", err)
	}

	shufflerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, keydesk.ShufflerPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("shuffler key: %w", err)
	}

	return routerPublicKey, shufflerPublicKey, nil
}
