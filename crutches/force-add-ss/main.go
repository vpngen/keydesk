package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"golang.org/x/crypto/nacl/box"
)

var (
	// ErrInvalidArgs - invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")
	// ErrProto0AlreadyPresent - Proto0 already presents.
	ErrProto0AlreadyPresent = errors.New("proto0 already presents")
	// ErrProto0AlreadyAbsent - Proto0 already absent.
	ErrProto0AlreadyAbsent = errors.New("proto0 already absent")
)

func main() {
	userId, key, brigadeID, dbDir, etcDir, addr, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	fmt.Fprintf(os.Stderr, "UserID: %s\n", userId)
	switch {
	case addr.IsValid() && !addr.Addr().IsUnspecified():
		fmt.Fprintf(os.Stderr, "Command address:port: %s\n", addr)
	case addr.IsValid():
		fmt.Fprintln(os.Stderr, "Command address:port is COMMON")
	default:
		fmt.Fprintln(os.Stderr, "Command address:port is for DEBUG")
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	if err = Do(db, &routerPublicKey, &shufflerPublicKey, userId, key); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (uuid.UUID, string, string, string, string, netip.AddrPort, error) {
	var (
		id            string
		dbdir, etcdir string
		err           error
		addrPort      netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return uuid.Nil, "", "", "", "", addrPort, fmt.Errorf("cannot define user: %w", err)
	}

	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	user := flag.String("u", "", "User ID")
	key := flag.String("k", "", "SS-key (base64)")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return uuid.Nil, "", "", "", "", addrPort, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return uuid.Nil, "", "", "", "", addrPort, fmt.Errorf("addr: %w", err)
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
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return uuid.Nil, "", "", "", "", addrPort, fmt.Errorf("etcdir dir: %w", err)
		}
	}

	userID, err := uuid.Parse(*user)
	if err != nil {
		return uuid.Nil, "", "", "", "", addrPort, fmt.Errorf("user: %w", err)
	}

	buf, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(*key)
	if err != nil {
		return uuid.Nil, "", "", "", "", addrPort, fmt.Errorf("key: %w", err)
	}

	_, secret, _ := strings.Cut(string(buf), ":")
	if secret == "" {
		return uuid.Nil, "", "", "", "", addrPort, ErrInvalidArgs
	}

	return userID, secret, id, dbdir, etcdir, addrPort, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte, userId uuid.UUID, key string) error {
	for {
		fullname, person, err := namesgenerator.ChemistryAwardeeShort()
		if err != nil {
			return fmt.Errorf("namesgenerator: %w", err)
		}

		vpnCfgs, err := db.GetVpnConfigs(nil)
		if err != nil {
			return fmt.Errorf("get vpn configs: %w", err)
		}

		if err := addUser(db, vpnCfgs, fullname, person, routerPublicKey, shufflerPublicKey, userId, key); err != nil {
			if errors.Is(err, storage.ErrUserCollision) {
				continue
			}

			return fmt.Errorf("addUser: %w", err)
		}

		return nil
	}
}

func addUser(
	db *storage.BrigadeStorage,
	vpnCfgs *storage.ConfigsImplemented,
	fullname string,
	person namesgenerator.Person,
	routerPublicKey,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	userID uuid.UUID,
	outlineSecret string,
) error {
	wgPub, _, _, wgRouterPSK, wgShufflerPSK, err := keydesk.GenUserWGKeys(routerPublicKey, shufflerPublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wg gen: %s\n", err)

		return fmt.Errorf("wg gen: %w", err)
	}

	var cloakByPassUIDRouterEnc, CloakByPassUIDShufflerEnc string

	if len(vpnCfgs.Ovc) > 0 || len(vpnCfgs.Outline) > 0 {
		var err error

		_, cloakByPassUIDRouterEnc, CloakByPassUIDShufflerEnc, err = keydesk.GenUserCloakKeys(routerPublicKey, shufflerPublicKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cloak gen: %s\n", err)

			return fmt.Errorf("ovc gen: %w", err)
		}
	}

	var (
		outlineSecretRouterEnc   string
		outlineSecretShufflerEnc string
	)
	if len(vpnCfgs.Outline) > 0 {
		outlineSecretRouterEnc, outlineSecretShufflerEnc, err = genUserOutlineSecret(routerPublicKey, shufflerPublicKey, outlineSecret)
		if err != nil {
			fmt.Fprintf(os.Stderr, "outline gen: %s\n", err)

			return fmt.Errorf("outline gen: %w", err)
		}
	}

	if _, err := db.CreateUser(
		userID,
		vpnCfgs, fullname, person,
		false, false,
		wgPub, wgRouterPSK, wgShufflerPSK,
		"", cloakByPassUIDRouterEnc, CloakByPassUIDShufflerEnc,
		"", "", "", "",
		outlineSecretRouterEnc, outlineSecretShufflerEnc,
		"", "",
	); err != nil {
		fmt.Fprintf(os.Stderr, "put: %s\n", err)

		return fmt.Errorf("put: %w", err)
	}

	return nil
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

func genUserOutlineSecret(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte, secret string) (string, string, error) {
	// TODO: why do we encrypt *encoded* secret?
	secretRouter, err := box.SealAnonymous(nil, []byte(secret), routerPublicKey, rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("secret router seal: %w", err)
	}

	secretShuffler, err := box.SealAnonymous(nil, []byte(secret), shufflerPublicKey, rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("secret shuffler seal: %w", err)
	}

	return base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretRouter),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretShuffler),
		nil
}
