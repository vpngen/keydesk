package main

import (
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
)

// Default web config.
const (
	DefaultHomeDir = ""
)

func parseArgs() (string, string, error) {
	brigadeID := flag.String("id", "", "brigadier_id") // !!! is id only for debug?
	homeDir := flag.String("d", DefaultHomeDir, "Dir for db files (for test)")

	flag.Parse()

	// brigadeID must be base32 decodable.
	id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(*brigadeID)
	if err != nil {
		return "", "", fmt.Errorf("id base32: %s: %w", *brigadeID, err)
	}

	_, err = uuid.FromBytes(id)
	if err != nil {
		return "", "", fmt.Errorf("id uuid: %s: %w", *brigadeID, err)
	}

	if *homeDir == "" {
		*homeDir = filepath.Join("home", *brigadeID)
	}

	dbdir, err := filepath.Abs(*homeDir)
	if err != nil {
		return "", "", fmt.Errorf("dbdir dir: %w", err)
	}

	return *brigadeID, dbdir, nil
}

func do() error {

	return nil
}

func main() {
	brigadeID, dbDir, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	db := &keydesk.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, keydesk.BrigadeFilename),
	}

	// just do it.
	err = keydesk.DestroyBrigade(db)
	if err != nil {
		log.Fatalf("Can't destroy brigade: %s", err)
	}

}
