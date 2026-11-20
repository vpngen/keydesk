// proto0-adopt gives brigade users the proto0 (VLESS) secret that their live xray
// client already uses, when brigade.json has none.
//
// Some brigade.json files lost the proto0 secrets of users whose xray clients the
// endpoint still serves; the cause is not known. Nothing shows until xray is rendered
// again from brigade.json (a replay or a migration), which silently drops those users
// (31.10.2026: 4,088 users on one server, 661 of them active).
//
// The endpoint keeps, per client, the email (the user's WireGuard public key), the id
// (the proto0 long ID) and the shortId (the proto0 short ID): exactly what the user's
// key carries. For every user WITHOUT a proto0 secret whose WireGuard public key
// matches a client, this seals "<shortId>-<id without dashes>" for the router and the
// shuffler as keydesk does at user creation. Nothing else is changed, no user that
// already has a secret is touched, and nothing is sent to the endpoint: its xray
// already serves these clients.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

// ErrInvalidArgs - invalid arguments.
var ErrInvalidArgs = errors.New("invalid arguments")

// wg-mng accepts a short ID of 8 to 16 hex digits.
var shortIDRe = regexp.MustCompile(`^[0-9a-fA-F]{8,16}$`)

// Client - one client of the endpoint's VLESS inbound.
type Client struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	ShortID string `json:"shortId"`
}

// Live - the endpoint's VLESS inbound settings, as
// jq -c '[.inbounds[]|select(.protocol=="vless")][0].settings' config.json prints them.
type Live struct {
	Clients []Client `json:"clients"`
}

// Report - what adopt did, or would do.
type Report struct {
	Users        int
	HadSecret    int
	Adopted      []string
	NoClient     int
	BadClients   []string
	UnusedClient int
}

func main() {
	input, brigadeID, dbDir, etcDir, dryRun, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)

	if dryRun {
		fmt.Fprintln(os.Stderr, "Dry run")
	}

	live, err := readLive(input)
	if err != nil {
		log.Fatalf("Can't read live clients: %s\n", err)
	}

	routerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(etcDir, keydesk.RouterPublicKeyFilename))
	if err != nil {
		log.Fatalf("Can't read router public key: %s\n", err)
	}

	shufflerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(etcDir, keydesk.ShufflerPublicKeyFilename))
	if err != nil {
		log.Fatalf("Can't read shuffler public key: %s\n", err)
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	if err := Do(db, live, &routerPublicKey, &shufflerPublicKey, dryRun); err != nil {
		log.Fatalf("Can't do: %s\n", err)
	}
}

func parseArgs() (string, string, string, string, bool, error) {
	sysUser, err := user.Current()
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("cannot define user: %w", err)
	}

	dryRun := flag.Bool("n", false, "Dry run: report, write nothing")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	etcDir := flag.String("c", keydesk.DefaultEtcDir, "Dir with "+keydesk.RouterPublicKeyFilename+" and "+keydesk.ShufflerPublicKeyFilename)

	flag.Parse()

	if flag.NArg() != 1 {
		return "", "", "", "", false, fmt.Errorf("%w: need one input file, or - for stdin", ErrInvalidArgs)
	}

	id := *brigadeID
	if id == "" {
		id = sysUser.Username
	}

	dbdir := *filedbDir
	switch {
	case dbdir != "":
		dbdir, err = filepath.Abs(dbdir)
		if err != nil {
			return "", "", "", "", false, fmt.Errorf("dbdir dir: %w", err)
		}
	case id == sysUser.Username:
		dbdir = filepath.Join(storage.DefaultHomeDir, id)
	default:
		dbdir, err = os.Getwd()
		if err != nil {
			return "", "", "", "", false, fmt.Errorf("cwd: %w", err)
		}
	}

	return flag.Arg(0), id, dbdir, *etcDir, *dryRun, nil
}

func readLive(name string) (*Live, error) {
	var r io.Reader = os.Stdin

	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		defer f.Close()

		r = f
	}

	live := &Live{}
	if err := json.NewDecoder(r).Decode(live); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return live, nil
}

// Do - adopt the live clients into brigade.json, keeping a copy of the file first.
func Do(db *storage.BrigadeStorage, live *Live, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte, dryRun bool) error {
	f, data, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer f.Close()

	rep, err := adopt(data, live, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return fmt.Errorf("adopt: %w", err)
	}

	for _, s := range rep.BadClients {
		fmt.Fprintf(os.Stderr, "Brigade %s: skipped client: %s\n", db.BrigadeID, s)
	}

	for _, s := range rep.Adopted {
		fmt.Fprintf(os.Stderr, "Brigade %s: user %s adopted\n", db.BrigadeID, s)
	}

	fmt.Printf("RESULT users=%d had_secret=%d adopted=%d no_client=%d unused_clients=%d bad_clients=%d\n",
		rep.Users, rep.HadSecret, len(rep.Adopted), rep.NoClient, rep.UnusedClient, len(rep.BadClients))

	if dryRun || len(rep.Adopted) == 0 {
		return nil
	}

	// The lock is held, so the copy is the exact file that gets replaced.
	orig, err := os.ReadFile(db.BrigadeFilename)
	if err != nil {
		return fmt.Errorf("read for backup: %w", err)
	}

	backup := db.BrigadeFilename + ".bak-proto0-adopt-" + time.Now().UTC().Format("20060102T150405")
	if err := os.WriteFile(backup, orig, 0o600); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Brigade %s: backup %s\n", db.BrigadeID, backup)

	if err := f.Commit(data); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Brigade %s: committed\n", db.BrigadeID)

	return nil
}

// adopt fills the proto0 secret of every user that has none and has a live client.
func adopt(data *storage.Brigade, live *Live, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (Report, error) {
	rep := Report{Users: len(data.Users)}

	byKey := make(map[string]Client, len(live.Clients))
	for _, c := range live.Clients {
		key, err := normKey(c.Email)
		if err != nil {
			rep.BadClients = append(rep.BadClients, fmt.Sprintf("email %q: %s", c.Email, err))

			continue
		}

		if _, err := uuid.Parse(c.ID); err != nil {
			rep.BadClients = append(rep.BadClients, fmt.Sprintf("email %q: id: %s", c.Email, err))

			continue
		}

		if !shortIDRe.MatchString(c.ShortID) {
			rep.BadClients = append(rep.BadClients, fmt.Sprintf("email %q: no valid shortId", c.Email))

			continue
		}

		if _, dup := byKey[key]; dup {
			return rep, fmt.Errorf("two clients for one key %q", c.Email)
		}

		byKey[key] = c
	}

	used := 0

	for _, u := range data.Users {
		if u.Proto0SecretRouterEnc != "" || u.Proto0SecretShufflerEnc != "" {
			rep.HadSecret++

			if _, ok := byKey[string(u.WgPublicKey)]; ok {
				used++
			}

			continue
		}

		c, ok := byKey[string(u.WgPublicKey)]
		if !ok {
			rep.NoClient++

			continue
		}

		used++

		secret := strings.ToLower(c.ShortID) + "-" + strings.ReplaceAll(strings.ToLower(c.ID), "-", "")

		routerEnc, err := box.SealAnonymous(nil, []byte(secret), routerPublicKey, rand.Reader)
		if err != nil {
			return rep, fmt.Errorf("secret router seal: %w", err)
		}

		shufflerEnc, err := box.SealAnonymous(nil, []byte(secret), shufflerPublicKey, rand.Reader)
		if err != nil {
			return rep, fmt.Errorf("secret shuffler seal: %w", err)
		}

		u.Proto0SecretRouterEnc = base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(routerEnc)
		u.Proto0SecretShufflerEnc = base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(shufflerEnc)

		rep.Adopted = append(rep.Adopted, fmt.Sprintf("%s (%s)", u.UserID, u.Name))
	}

	rep.UnusedClient = len(byKey) - used

	return rep, nil
}

// normKey decodes a WireGuard public key written as standard or URL-safe base64, with
// or without padding, into the raw bytes brigade.json keeps.
func normKey(s string) (string, error) {
	s = strings.NewReplacer("-", "+", "_", "/").Replace(strings.TrimRight(s, "="))

	b, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("not base64: %w", err)
	}

	if len(b) != 32 {
		return "", fmt.Errorf("not a 32-byte key (%d bytes)", len(b))
	}

	return string(b), nil
}
