package main

import (
	"bufio"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/netip"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
)

// ErrInvalidArgs is returned when arguments are invalid.
var ErrInvalidArgs = errors.New("invalid arguments")

func main() {
	dryrun, brigadeID, dbDir, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)

	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     netip.AddrPort{},
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	m, err := readMap(os.Stdin)
	if err != nil {
		log.Fatalf("Read map: %s\n", err)
	}

	if err := Do(db, m, dryrun); err != nil {
		log.Fatalf("Do: %s\n", err)
	}
}

// Do re-encodes wg private keys.
func Do(db *storage.BrigadeStorage, m map[string]map[netip.Addr]string, dryrun bool) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	bn := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(data.WgPublicKey)

	b, ok := m[bn]
	if !ok {
		fmt.Fprintf(os.Stderr, "Brigade not found: %s\n", bn)

		return nil
	}

	for _, user := range data.Users {
		fmt.Fprintf(os.Stderr, "User: %s\n", user.Name)

		u, ok := b[user.IPv4Addr]
		if !ok {
			fmt.Fprintf(os.Stderr, "User not found: %s\n", user.IPv4Addr)

			continue
		}

		if !dryrun {
			pub, err := base64.StdEncoding.WithPadding(base64.StdPadding).DecodeString(u)
			if err != nil {
				return fmt.Errorf("decode: %w", err)
			}

			user.WgPublicKey = pub
		}
	}

	if !dryrun {
		f.Commit(data)
	}

	return nil
}

func parseArgs() (bool, string, string, error) {
	var (
		id    string
		dbdir string
		err   error
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, "", "", fmt.Errorf("cannot define user: %w", err)
	}

	dryrun := flag.Bool("n", false, "dry run")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, "", "", fmt.Errorf("dbdir dir: %w", err)
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

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return false, "", "", fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return false, "", "", fmt.Errorf("id uuid: %s: %w", id, err)
	}

	return *dryrun, id, dbdir, nil
}

func readMap(r io.Reader) (map[string]map[netip.Addr]string, error) {
	m := make(map[string]map[netip.Addr]string)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		encUserPub, line, _ := strings.Cut(line, ":")
		if encUserPub == "" || line == "" {
			fmt.Fprintf(os.Stderr, "Invalid line: %s\n", line)

			continue
		}

		ipStr, encBrigadePub, _ := strings.Cut(line, ":")

		brigadePub, err := url.QueryUnescape(encBrigadePub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid Brigade: %s\n", encBrigadePub)

			continue
		}

		userPub, err := url.QueryUnescape(encUserPub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid User: %s\n", encUserPub)

			continue
		}

		ip, err := netip.ParseAddr(ipStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid IP: %s\n", ipStr)

			continue
		}

		b, ok := m[brigadePub]
		if !ok {
			b = make(map[netip.Addr]string)
			m[brigadePub] = b
		}

		b[ip] = userPub
	}

	return m, nil
}
