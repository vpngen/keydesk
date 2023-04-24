package main

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

// ErrInvalidArgs is returned when arguments are invalid.
var ErrInvalidArgs = errors.New("invalid arguments")

func main() {
	dryrun, brigadeID, dbDir, etcDir, oldKeyFilename, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	oldRouterPrivateKey, oldRouterPublicKey, err := readPrivKey(oldKeyFilename)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	fmt.Fprintf(os.Stderr, "EtcDir: %s\n", etcDir)
	fmt.Fprintf(os.Stderr, "OldKeyFilename: %s\n", oldKeyFilename)

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

	if err := Do(db, dryrun, &routerPublicKey, &shufflerPublicKey, &oldRouterPrivateKey, &oldRouterPublicKey); err != nil {
		log.Fatalf("Do: %s\n", err)
	}
}

// Do re-encodes wg private keys.
func Do(db *storage.BrigadeStorage, dryrun bool, routerPublicKey, shufflerPublicKey, oldRouterPrivateKey, oldRouterPublicKey *[naclkey.NaclBoxKeyLength]byte) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	ok, routerEnc, shufflerEnc, err := reEncode(data.WgPrivateRouterEnc, routerPublicKey, shufflerPublicKey, oldRouterPrivateKey, oldRouterPublicKey)
	if err != nil {
		return fmt.Errorf("wg private router: %w", err)
	}

	switch ok {
	case true:
		fmt.Fprintln(os.Stderr, "*[!] WgPrivateRouterEnc was reencoded")

		if !dryrun {
			data.WgPrivateRouterEnc = routerEnc
			data.WgPrivateShufflerEnc = shufflerEnc
		}
	default:
		fmt.Fprintln(os.Stderr, " [=] WgPrivateRouterEnc was not reencoded")
	}

	for _, user := range data.Users {
		ok, routerEnc, shufflerEnc, err := reEncode(user.WgPSKRouterEnc, routerPublicKey, shufflerPublicKey, oldRouterPrivateKey, oldRouterPublicKey)
		if err != nil {
			return fmt.Errorf("wg private router: %w", err)
		}

		switch ok {
		case true:
			fmt.Fprintf(os.Stderr, "*[!] WgPSKRouterEnc for user %s was reencoded\n", base32.StdEncoding.EncodeToString(user.WgPublicKey[:]))

			if !dryrun {
				user.WgPSKRouterEnc = routerEnc
				user.WgPSKShufflerEnc = shufflerEnc
			}

		default:
			fmt.Fprintf(os.Stderr, " [=] WgPSKRouterEnc for user %s was not reencoded\n", base32.StdEncoding.EncodeToString(user.WgPublicKey[:]))
		}
	}

	f.Commit(data)

	return nil
}

func reEncode(payload []byte, routerPublicKey, shufflerPublicKey, oldRouterPrivateKey, oldRouterPublicKey *[naclkey.NaclBoxKeyLength]byte) (bool, []byte, []byte, error) {
	if decrypted, ok := box.OpenAnonymous(nil, payload, oldRouterPublicKey, oldRouterPrivateKey); ok {
		// re-encode
		routerReEncrypted, err := box.SealAnonymous(nil, decrypted, routerPublicKey, rand.Reader)
		if err != nil {
			return false, nil, nil, fmt.Errorf("router seal: %w", err)
		}

		shufflerReEncrypted, err := box.SealAnonymous(nil, decrypted, shufflerPublicKey, rand.Reader)
		if err != nil {
			return false, nil, nil, fmt.Errorf("router seal: %w", err)
		}

		return true, routerReEncrypted, shufflerReEncrypted, nil
	}

	return false, nil, nil, nil
}

func parseArgs() (bool, string, string, string, string, error) {
	var (
		id                     string
		dbdir, etcdir, keypath string
		err                    error
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, "", "", "", "", fmt.Errorf("cannot define user: %w", err)
	}

	dryrun := flag.Bool("n", false, "dry run")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	oldRouterPrivKey := flag.String("r", "", "Old private router key")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, "", "", "", "", fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return false, "", "", "", "", fmt.Errorf("etcdir dir: %w", err)
		}
	}

	if *oldRouterPrivKey != "" {
		keypath, err = filepath.Abs(*oldRouterPrivKey)
		if err != nil {
			return false, "", "", "", "", fmt.Errorf("router key: %w", err)
		}
	}

	switch *brigadeID {
	case "", sysUser.Username:
		id = sysUser.Username

		if *filedbDir == "" {
			dbdir = filepath.Join(storage.DefaultHomeDir, id)
		}

		if *etcDir == "" {
			etcdir = keydesk.DefaultEtcDir
		}

		if *oldRouterPrivKey == "" {
			keypath = ""
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

		if *etcDir == "" {
			etcdir = cwd
		}

		if *oldRouterPrivKey == "" {
			keypath = ""
		}
	}

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return false, "", "", "", "", fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return false, "", "", "", "", fmt.Errorf("id uuid: %s: %w", id, err)
	}

	return *dryrun, id, dbdir, etcdir, keypath, nil
}

func readPubKeys(path string) ([naclkey.NaclBoxKeyLength]byte, [naclkey.NaclBoxKeyLength]byte, error) {
	empty := [naclkey.NaclBoxKeyLength]byte{}

	routerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, keydesk.RouterPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("router key: %w", err)
	}

	shufflerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, keydesk.ShufflerPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("shuffler key: %w", err)
	}

	return routerPublicKey, shufflerPublicKey, nil
}

func readPrivKey(filename string) ([naclkey.NaclBoxKeyLength]byte, [naclkey.NaclBoxKeyLength]byte, error) {
	empty := [naclkey.NaclBoxKeyLength]byte{}

	if filename == "" {
		if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
			blob, err := io.ReadAll(os.Stdin)
			if err != nil {
				return empty, empty, fmt.Errorf("read: %w", err)
			}

			keys, err := naclkey.UnmarshalKeypair(blob)
			if err != nil {
				return empty, empty, fmt.Errorf("parse key: %w", err)
			}

			return keys.Private, keys.Public, nil
		}
	}

	keys, err := naclkey.ReadKeypairFile(filename)
	if err != nil {
		return empty, empty, fmt.Errorf("old router key: %w", err)
	}

	return keys.Private, keys.Public, nil
}
