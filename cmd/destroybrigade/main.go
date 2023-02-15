package main

import (
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os/user"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
)

// Default web config.
const (
	DefaultHomeDir = ""
)

const anyAddrport = "0.0.0.0:0"

func parseArgs() (netip.AddrPort, string, string, error) {
	addrPort := netip.AddrPort{}

	// is id only for debug?
	brigadeID := flag.String("id", "", "brigadier_id")
	homeDir := flag.String("d", DefaultHomeDir, "Dir for db files (for test)")
	addr := flag.String("a", anyAddrport, "API endpoint address:port")

	flag.Parse()

	if *brigadeID == "" {
		username, err := user.Current()
		if err != nil {
			return addrPort, "", "", fmt.Errorf("username: %w", err)
		}

		brigadeID = &username.Username
	}

	// brigadeID must be base32 decodable.
	id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(*brigadeID)
	if err != nil {
		return addrPort, "", "", fmt.Errorf("id base32: %s: %w", *brigadeID, err)
	}

	_, err = uuid.FromBytes(id)
	if err != nil {
		return addrPort, "", "", fmt.Errorf("id uuid: %s: %w", *brigadeID, err)
	}

	if *homeDir == "" {
		*homeDir = filepath.Join("home", *brigadeID)
	}

	dbdir, err := filepath.Abs(*homeDir)
	if err != nil {
		return addrPort, "", "", fmt.Errorf("dbdir dir: %w", err)
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return addrPort, "", "", fmt.Errorf("api addr: %w", err)
		}
	}

	return addrPort, *brigadeID, dbdir, nil
}

func do() error {

	return nil
}

func main() {
	addr, brigadeID, dbDir, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	db := &keydesk.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, keydesk.BrigadeFilename),
		APIAddrPort:     addr,
	}

	// just do it.
	err = keydesk.DestroyBrigade(db)
	if err != nil {
		log.Fatalf("Can't destroy brigade: %s", err)
	}

}
