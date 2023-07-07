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

	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
)

var (
	// ErrInvalidArgs - invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")

	// ErrInvalidPort - invalid port.
	ErrInvalidPort = errors.New("invalid port")

	// ErrInvalidDomainName - invalid domain name.
	ErrInvalidDomainName = errors.New("invalid domain name")
)

func main() {
	brigadeID, dbDir, addr, domain, port, err := parseArgs()
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

	if domain != "." {
		fmt.Fprintf(os.Stderr, "Domain set to: %q\n", domain)
	}

	if port != -1 {
		fmt.Fprintf(os.Stderr, "Port set to: %d\n", port)
	}

	if domain == "." && port == -1 {
		fmt.Fprintln(os.Stderr, "Domain and port was keept. Do nothing.")

		return
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

	if err = Do(db, addr, domain, port); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (string, string, netip.AddrPort, string, int, error) {
	var (
		id       string
		dbdir    string
		err      error
		addrPort netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return "", "", addrPort, "", 0, fmt.Errorf("cannot define user: %w", err)
	}

	port := flag.Int("p", -1, "Port, 0 - for random, -1 - for keep")
	domainName := flag.String("dn", ".", "domainName, empty - for empty, '.' - for keep")

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")

	flag.Parse()

	endpointPort := *port
	if endpointPort == 0 {
		endpointPort = int(rand.Int31n(keydesk.HighWireguardPort-keydesk.LowWireguardPort) + keydesk.LowWireguardPort)
	}

	if endpointPort != -1 && endpointPort <= keydesk.LowLimitPort {
		return "", "", addrPort, "", 0, fmt.Errorf("port: %d: %w", endpointPort, ErrInvalidPort)
	}

	endpointDomain := *domainName
	if endpointDomain != "." && endpointDomain != "" && !kdlib.IsDomainNameValid(endpointDomain) {
		return "", "", addrPort, "", 0, fmt.Errorf("domain: %s: %w", endpointDomain, ErrInvalidDomainName)
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return "", "", addrPort, "", 0, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return "", "", addrPort, "", 0, fmt.Errorf("addr: %w", err)
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

	return id, dbdir, addrPort, endpointDomain, endpointPort, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, addr netip.AddrPort, domain string, port int) error {
	if domain != "." {
		if err := db.DomainSet(domain); err != nil {
			return fmt.Errorf("set domain: %w", err)
		}
	}

	if port != -1 {
		if err := db.PortSet(uint16(port)); err != nil {
			return fmt.Errorf("set port: %w", err)
		}

		if err := db.ReplayBrigade(true, false, false); err != nil {
			return fmt.Errorf("replay: %w", err)
		}
	}

	return nil
}
