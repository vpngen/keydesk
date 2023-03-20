package main

import (
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
)

const random_data_env = "VGSTATS_RANDOM_DATA"

func main() {
	var rdata bool

	//  dbDir, statsDir, err
	addr, brigadeID, dbDir, statsDir, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	fmt.Fprintf(os.Stderr, "Statistics dir: %s\n", statsDir)
	switch {
	case addr.IsValid() && !addr.Addr().IsUnspecified():
		fmt.Fprintf(os.Stderr, "Command address:port: %s\n", addr)
	case addr.IsValid():
		fmt.Fprintln(os.Stderr, "Command address:port is COMMON")
	default:
		fmt.Fprintln(os.Stderr, "Command address:port is for DEBUG")
		if os.Getenv(random_data_env) != "" {
			rdata = true
			fmt.Fprintln(os.Stderr, "Random data is ON")
		}
	}

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	done := make(chan struct{})
	kill := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit

		fmt.Fprintln(os.Stderr, "Stopping...")

		close(kill)
	}()

	fmt.Fprintln(os.Stderr, "Starting...")

	go CollectingData(kill, done, addr, rdata, brigadeID, dbDir, statsDir)

	<-done
}

func parseArgs() (netip.AddrPort, string, string, string, error) {
	var (
		id              string
		dbdir, statsdir string
		addrPort        netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return addrPort, "", "", "", fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	statsDir := flag.String("s", "", "Dir with brigades statistics. Default: "+storage.DefaultStatsDir+"/<BrigadeID>")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return addrPort, "", "", "", fmt.Errorf("home base dir: %w", err)
		}
	}

	if *statsDir != "" {
		statsdir, err = filepath.Abs(*statsDir)
		if err != nil {
			return addrPort, "", "", "", fmt.Errorf("stats dir: %w", err)
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
