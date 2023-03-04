package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBrigadesListFileCheckDuration = 30 * time.Second
	DefaultStatisticsFetchingDuration    = 60 * time.Second
	DefaultJitterValue                   = 10 // sec
)

type BrigadeListRecord struct {
	BrigadeID string
	Addr      netip.Addr
	WgPub     []byte
}

type BrigadeListMap map[string]*BrigadeListRecord

type BrigadeQuotaJob struct {
	BrigadeID string
	Kill      chan struct{}
}

var (
	ErrInvalidListFormat = errors.New("invalid list format")
)

func NewJob(brigadeID string) *BrigadeQuotaJob {
	return &BrigadeQuotaJob{
		BrigadeID: brigadeID,
		Kill:      make(chan struct{}),
	}
}

type BrigadeQuotaJobsList struct {
	M                map[string]*BrigadeQuotaJob
	WG               *sync.WaitGroup
	Debug            bool
	QuotasBaseDir    string
	BrigadesListFile string
}

func NewJobList(debug bool, quotasDir, listFile string) *BrigadeQuotaJobsList {
	return &BrigadeQuotaJobsList{
		M:                make(map[string]*BrigadeQuotaJob),
		WG:               &sync.WaitGroup{},
		Debug:            debug,
		QuotasBaseDir:    quotasDir,
		BrigadesListFile: listFile,
	}
}

func (l BrigadeQuotaJobsList) Add(brigadeID string, addr netip.Addr, wgPub []byte) {
	if _, ok := l.M[brigadeID]; !ok {
		job := NewJob(brigadeID)
		l.M[brigadeID] = job

		l.WG.Add(1)

		go collectingData(l.Debug, l.WG, job.Kill, brigadeID, l.QuotasBaseDir, addr, wgPub)
	}
}

func (l BrigadeQuotaJobsList) Remove(brigadeID string) {
	if job, ok := l.M[brigadeID]; ok {
		close(job.Kill)
	}

	delete(l.M, brigadeID)
}

func (l BrigadeQuotaJobsList) RemoveAll() {
	for brigadeID := range l.M {
		l.Remove(brigadeID)
	}
}

func (l BrigadeQuotaJobsList) Refresh(kill <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	defer l.WG.Wait()
	defer l.RemoveAll()

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
			fmt.Fprintf(
				os.Stderr,
				"Add new brigade: %s addr: %s key: %s\n",
				brigadeID,
				r.Addr,
				base64.StdEncoding.Strict().WithPadding(base64.StdPadding).EncodeToString(r.WgPub),
			)
		}
	}
}

func (l BrigadeQuotaJobsList) removeDeleted(list BrigadeListMap) {
	for brigadeID, _ := range l.M {
		if _, ok := list[brigadeID]; !ok {
			l.Remove(brigadeID)
			fmt.Fprintf(
				os.Stderr,
				"Remove deleted brigade: %s\n",
				brigadeID,
			)
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
