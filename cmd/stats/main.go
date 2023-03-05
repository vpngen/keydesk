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
	// debug, workDir, dbDir, quotasDir, listDir, err
	debug, onedir, fakeaddr, workDir, dbBaseDir, quotasBaseDir, statsDir, listFile, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "DB Base Dir: %s\n", dbBaseDir)
	fmt.Fprintf(os.Stderr, "Quotas Base Dir: %s\n", quotasBaseDir)
	fmt.Fprintf(os.Stderr, "Brigades list file: %s\n", listFile)
	if debug {
		fmt.Fprintln(os.Stderr, "DEBUG mode")

		if onedir {
			fmt.Fprintln(os.Stderr, "ONE DIR mode")
		}

		if fakeaddr {
			fmt.Fprintln(os.Stderr, "Fake API calls mode")
		}
	}

	poller := NewJobList(debug, onedir, fakeaddr, workDir, dbBaseDir, quotasBaseDir, statsDir, listFile)

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

func parseArgs() (bool, bool, bool, string, string, string, string, string, error) {
	var (
		workdir, hbasedir, qbasedir, statsdir, listfile string
		err                                             error
	)

	homeBaseDir := flag.String("b", "", "Dir base for dirs with brigade database. Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	quotasBaseDir := flag.String("q", "", "Dir base for dirs with quotas and traffic statistics. Default: "+storage.DefaultQuotasDir+"/<BrigadeID>")
	statsDir := flag.String("s", "", "Dir with brigades statistics. Default: "+storage.DefaultStatsDir)
	workDir := flag.String("w", "", "Dir with working files. Default: "+DefultWorkingDir)
	listFile := flag.String("l", "", "Brigades list file. Default: "+filepath.Join(keydesk.DefaultBrigadesListDir, keydesk.DefaultBrigadesListFile))
	oneDir := flag.Bool("o", false, "Don't create separate subdirs for brigades (only with -d)")
	debug := flag.Bool("d", false, "Debug")
	fakeAddr := flag.Bool("a", false, "Fake API calls (only with -d)")

	flag.Parse()

	if *workDir != "" {
		qbasedir, err = filepath.Abs(*workDir)
		if err != nil {
			return false, false, false, "", "", "", "", "", fmt.Errorf("work dir: %w", err)
		}
	}

	if *homeBaseDir != "" {
		hbasedir, err = filepath.Abs(*homeBaseDir)
		if err != nil {
			return false, false, false, "", "", "", "", "", fmt.Errorf("home base dir: %w", err)
		}
	}

	if *quotasBaseDir != "" {
		qbasedir, err = filepath.Abs(*quotasBaseDir)
		if err != nil {
			return false, false, false, "", "", "", "", "", fmt.Errorf("quota base dir: %w", err)
		}
	}

	if *statsDir != "" {
		qbasedir, err = filepath.Abs(*statsDir)
		if err != nil {
			return false, false, false, "", "", "", "", "", fmt.Errorf("stats dir: %w", err)
		}
	}

	if *listFile != "" {
		listfile, err = filepath.Abs(*listFile)
		if err != nil {
			return false, false, false, "", "", "", "", "", fmt.Errorf("brigades list: %w", err)
		}
	}

	switch *debug {
	case true:
		cwd, err := os.Getwd()
		if err == nil {
			cwd, _ = filepath.Abs(cwd)
		}

		if *workDir == "" {
			workdir = cwd
		}

		if *homeBaseDir == "" {
			hbasedir = cwd
		}

		if *quotasBaseDir == "" {
			qbasedir = cwd
		}

		if *statsDir == "" {
			statsdir = cwd
		}

		if *listFile == "" {
			listfile = filepath.Join(cwd, keydesk.DefaultBrigadesListFile)
		}
	default:
		*oneDir = false
		*fakeAddr = false

		if *workDir == "" {
			workdir = DefultWorkingDir
		}

		if *homeBaseDir == "" {
			qbasedir = storage.DefaultHomeDir
		}

		if *quotasBaseDir == "" {
			qbasedir = storage.DefaultQuotasDir
		}

		if *statsDir == "" {
			qbasedir = storage.DefaultStatsDir
		}

		if *listFile == "" {
			listfile = filepath.Join(keydesk.DefaultBrigadesListDir, keydesk.DefaultBrigadesListFile)
		}
	}

	return *debug, *oneDir, *fakeAddr, workdir, hbasedir, qbasedir, statsdir, listfile, nil
}
