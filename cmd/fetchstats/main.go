package main

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk/storage"
)

const AggrStatsVersion = 1

type AggrStats struct {
	Ver   int              `json:"version"`
	Stats []*storage.Stats `json:"stats"`
}

var (
	ErrEmptyBrigadeName = errors.New("empty name")
	ErrEmptyBrigadeList = errors.New("empty list")
)

func main() {
	var w io.WriteCloser

	chunked, statsBaseDir, brigades, err := parseArgs()
	if err != nil {
		log.Fatalf("Invalid flags: %s\n", err)
	}

	stats, err := getStats(statsBaseDir, brigades)
	if err != nil {
		log.Fatalf("Can't harvest: %s\n", err)
	}

	switch chunked {
	case true:
		w = httputil.NewChunkedWriter(os.Stdout)
		defer w.Close()
	default:
		w = os.Stdout
	}

	if _, err := w.Write(stats); err != nil {
		log.Fatalf("Print stats: %s", err)
	}

}

func getStats(statsBaseDir string, brigades []string) ([]byte, error) {
	astats := &AggrStats{
		Ver: AggrStatsVersion,
	}

	for _, id := range brigades {
		buf, err := os.ReadFile(filepath.Join(statsBaseDir, id, storage.StatsFilename))
		if err != nil {
			continue
		}

		stats := &storage.Stats{}
		if err := json.Unmarshal(buf, stats); err != nil {
			continue
		}

		astats.Stats = append(astats.Stats, stats)
	}

	buf, err := json.MarshalIndent(astats, " ", " ")
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf, nil
}

func parseArgs() (bool, string, []string, error) {
	var (
		statsdir string
	)

	// is id only for debug?
	chunked := flag.Bool("ch", false, "chunked output")
	brigadesList := flag.String("b", "", "Brigaders list by commas")
	statsBaseDir := flag.String("s", storage.DefaultStatsDir, "Dir base for dirs brigades statistics.")

	flag.Parse()

	statsdir, err := filepath.Abs(*statsBaseDir)
	if err != nil {
		return false, "", nil, fmt.Errorf("statsdir dir: %w", err)
	}

	if *brigadesList == "" {
		return false, "", nil, fmt.Errorf("brigades list: %w", ErrEmptyBrigadeList)
	}

	brigades := strings.Split(*brigadesList, ",")
	for _, id := range brigades {
		if id == "" {
			return false, "", nil, fmt.Errorf("brigade id: %w", ErrEmptyBrigadeName)
		}

		// brigadeID must be base32 decodable.
		binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
		if err != nil {
			return false, "", nil, fmt.Errorf("id base32: %s: %w", id, err)
		}

		_, err = uuid.FromBytes(binID)
		if err != nil {
			return false, "", nil, fmt.Errorf("id uuid: %s: %w", id, err)
		}
	}

	return *chunked, statsdir, brigades, nil
}