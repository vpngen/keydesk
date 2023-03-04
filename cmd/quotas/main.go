package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func main() {
	// debug, quotaDir, listDir, err
	debug, quotasDir, listFile, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "Quotas Dir: %s\n", quotasDir)
	fmt.Fprintf(os.Stderr, "Brigades list file: %s\n", listFile)
	if debug {
		fmt.Fprintln(os.Stderr, "DEBUG mode")
	}

	poller := NewJobList(debug, quotasDir, listFile)

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

	go poller.Refresh(kill, done)

	<-done
}

func parseArgs() (bool, string, string, error) {
	var (
		quotasdir, listfile string
		err                 error
	)

	quotasDir := flag.String("q", "", "Dir with quotas and traffic statistics. Default: "+storage.DefaultQuotasDir)
	listFile := flag.String("l", "", "Brigades list file. Default: "+filepath.Join(keydesk.DefaultBrigadesListDir, keydesk.DefaultBrigadesListFile))
	debug := flag.Bool("d", false, "Debug")

	flag.Parse()

	if *quotasDir != "" {
		quotasdir, err = filepath.Abs(*quotasDir)
		if err != nil {
			return false, "", "", fmt.Errorf("statdir dir: %w", err)
		}
	}

	if *listFile != "" {
		listfile, err = filepath.Abs(*listFile)
		if err != nil {
			return false, "", "", fmt.Errorf("statdir dir: %w", err)
		}
	}

	switch *debug {
	case true:
		cwd, err := os.Getwd()
		if err == nil {
			cwd, _ = filepath.Abs(cwd)
		}

		if *quotasDir == "" {
			quotasdir = cwd
		}

		if *listFile == "" {
			listfile = filepath.Join(cwd, keydesk.DefaultBrigadesListFile)
		}
	default:
		if *quotasDir == "" {
			quotasdir = storage.DefaultQuotasDir
		}

		if *listFile == "" {
			listfile = filepath.Join(keydesk.DefaultBrigadesListDir, keydesk.DefaultBrigadesListFile)
		}
	}

	return *debug, quotasdir, listfile, nil
}
