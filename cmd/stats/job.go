package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vpngen/keydesk/keydesk/storage"
)

type BrigadeListRecord struct {
	BrigadeID string
	Addr      netip.Addr
	WgPub     []byte
}

type BrigadeListMap map[string]*BrigadeListRecord

type BrigadeQuotaJob struct {
	BrigadeID       string
	WorkingFilename string
	QuotasFilename  string
	StatsFilename   string
	Kill            chan struct{}
	Done            chan struct{}
}

var (
	ErrInvalidListFormat = errors.New("invalid list format")
)

func NewJob(brigadeID, workFilename, quotasFilename, statsFilename string) *BrigadeQuotaJob {
	return &BrigadeQuotaJob{
		BrigadeID:       brigadeID,
		WorkingFilename: workFilename,
		QuotasFilename:  quotasFilename,
		StatsFilename:   statsFilename,
		Kill:            make(chan struct{}),
		Done:            make(chan struct{}),
	}
}

type BrigadeQuotaJobsList struct {
	M                map[string]*BrigadeQuotaJob
	WG               *sync.WaitGroup
	Debug            bool
	OneDir           bool
	FakeAddress      bool
	WorkingDir       string
	HomeBaseDir      string
	QuotasBaseDir    string
	StatsDir         string
	BrigadesListFile string
}

func NewJobList(debug, onedir, fakeaddr bool, workDir, homeBaseDir, quotasBaseDir, statsDir, listFile string) *BrigadeQuotaJobsList {
	return &BrigadeQuotaJobsList{
		M:                make(map[string]*BrigadeQuotaJob),
		WG:               &sync.WaitGroup{},
		Debug:            debug,
		OneDir:           onedir,
		FakeAddress:      fakeaddr,
		WorkingDir:       workDir,
		HomeBaseDir:      homeBaseDir,
		QuotasBaseDir:    quotasBaseDir,
		StatsDir:         statsDir,
		BrigadesListFile: listFile,
	}
}

func (l BrigadeQuotaJobsList) Add(brigadeID string, addr netip.Addr, wgPub []byte) {
	if !l.OneDir {
		if _, err := os.Stat(filepath.Join(l.QuotasBaseDir, brigadeID)); os.IsNotExist(err) {
			return
		}
	}

	if _, ok := l.M[brigadeID]; !ok {
		brigadeFilename := filepath.Join(l.HomeBaseDir, brigadeID, storage.BrigadeFilename)
		if l.OneDir {
			brigadeFilename = filepath.Join(l.HomeBaseDir, brigadeID+"-"+storage.BrigadeFilename)
		}

		counterFilename := filepath.Join(l.HomeBaseDir, brigadeID, storage.KeydeskCountersFilename)
		if l.OneDir {
			counterFilename = filepath.Join(l.HomeBaseDir, brigadeID+"-"+storage.KeydeskCountersFilename)
		}

		quotasFilename := filepath.Join(l.QuotasBaseDir, brigadeID, storage.QuotasFilename)
		if l.OneDir {
			quotasFilename = filepath.Join(l.QuotasBaseDir, brigadeID+"-"+storage.QuotasFilename)
		}

		statsFilename := filepath.Join(l.StatsDir, brigadeID+"-"+storage.StatsFilename)
		workFilename := filepath.Join(l.WorkingDir, brigadeID+"-"+WorkingFilename)

		job := NewJob(brigadeID, workFilename, quotasFilename, statsFilename)
		l.M[brigadeID] = job

		fmt.Fprintf(
			os.Stderr,
			"Add new brigade: %s addr: %s key: %s\n",
			brigadeID,
			addr,
			base64.StdEncoding.Strict().WithPadding(base64.StdPadding).EncodeToString(wgPub),
		)

		l.WG.Add(1)

		go collectingData(l.Debug, l.FakeAddress, l.WG, job.Kill, job.Done, brigadeID, workFilename, brigadeFilename, counterFilename, quotasFilename, statsFilename, addr, wgPub)
	}
}

func (l BrigadeQuotaJobsList) Remove(brigadeID string) {
	if job, ok := l.M[brigadeID]; ok {
		close(job.Kill)

		fmt.Fprintf(
			os.Stderr,
			"Remove deleted brigade: %s\n",
			brigadeID,
		)

		<-job.Done

		os.Remove(job.WorkingFilename)
		os.Remove(job.QuotasFilename)
		os.Remove(job.StatsFilename)

		if !l.OneDir {
			os.Remove(filepath.Join(l.QuotasBaseDir, brigadeID))
		}
	}

	delete(l.M, brigadeID)
}

func (l BrigadeQuotaJobsList) FinishAll() {
	for _, job := range l.M {
		close(job.Kill)
	}
}

func (l BrigadeQuotaJobsList) Refresh(kill <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	defer l.WG.Wait()
	defer l.FinishAll()

	timer := time.NewTimer(time.Second)

	fi := os.FileInfo(nil)

	for {
		select {
		case ts := <-timer.C:
			fmt.Fprintf(os.Stderr, "%s: Checking list...\n", ts.UTC().Format(time.RFC3339))

			fi = l.refresh(fi)

			timer.Reset(DefaultBrigadesListFileCheckDuration)
		case <-kill:
			if !timer.Stop() {
				<-timer.C
			}

			return
		}
	}
}

func (l BrigadeQuotaJobsList) refresh(prevFi os.FileInfo) os.FileInfo {
	fi, err := os.Stat(l.BrigadesListFile)
	if err != nil {
		return nil
	}

	if prevFi != nil {
		if prevFi.ModTime().Equal(fi.ModTime()) {
			return prevFi
		}
	}

	newList, err := readBrigadesList(l.BrigadesListFile)
	if err != nil {
		return nil
	}

	l.addNew(newList)
	l.removeDeleted(newList)

	return fi
}

func (l BrigadeQuotaJobsList) addNew(list BrigadeListMap) {
	for brigadeID, r := range list {
		if _, ok := l.M[brigadeID]; !ok {
			l.Add(brigadeID, r.Addr, r.WgPub)
		}
	}
}

func (l BrigadeQuotaJobsList) removeDeleted(list BrigadeListMap) {
	for brigadeID := range l.M {
		if _, ok := list[brigadeID]; !ok {
			l.Remove(brigadeID)
		}
	}
}

func readBrigadesList(filename string) (BrigadeListMap, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	defer f.Close()

	list := BrigadeListMap{}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		record := strings.Split(line, ";")
		if len(record) != 3 || record[0] == "" {
			return nil, fmt.Errorf("%w", ErrInvalidListFormat)
		}

		addr, err := netip.ParseAddr(record[1])
		if err != nil {
			return nil, fmt.Errorf("addr: %w", err)
		}

		wgkey, err := base64.StdEncoding.DecodeString(record[2])
		if err != nil {
			return nil, fmt.Errorf("wgkey: %w", err)
		}

		list[record[0]] = &BrigadeListRecord{
			BrigadeID: record[0],
			Addr:      addr,
			WgPub:     wgkey,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	return list, nil
}
