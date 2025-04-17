package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"
	"sort"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
)

var (
	// ErrInvalidArgs - invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")

	// ErrNeedFullReplay - need full replay.
	ErrNeedFullReplay = errors.New("need full replay")

	// ErrNeedRestart - need restart.
	ErrNeedRestart = errors.New("need restart")
)

func main() {
	newfile, brigadeID, dbDir, addr, dryRun, replay, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Brigade: %s\n", brigadeID)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	switch {
	case addr.IsValid() && !addr.Addr().IsUnspecified():
		fmt.Fprintf(os.Stderr, "Command address:port: %s\n", addr)
	case addr.IsValid():
		fmt.Fprintln(os.Stderr, "Command address:port is COMMON")
	default:
		fmt.Fprintln(os.Stderr, "Command address:port is for DEBUG")
	}

	if dryRun {
		fmt.Fprintln(os.Stderr, "Dry run")
	}

	if replay && !dryRun {
		fmt.Fprintln(os.Stderr, "Replay")
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

	errDo := Do(db, newfile, dryRun)
	if errDo != nil &&
		!errors.Is(errDo, ErrNeedRestart) &&
		!errors.Is(errDo, ErrNeedFullReplay) {
		log.Fatalf("Can't do: %s\n", err)
	}

	if !dryRun && replay {
		switch {
		case errors.Is(errDo, ErrNeedRestart):
			fmt.Fprintln(os.Stderr, "Need restart")
			os.Exit(2)
		case errors.Is(errDo, ErrNeedFullReplay):
			fmt.Fprintln(os.Stderr, "Need full replay. Do it")
			if err := db.ReplayBrigade(true, false, false, true, false); err != nil {
				log.Fatalf("replay brigade: %s", err)
			}
		case errDo != nil:
			log.Fatalf("Can't do: %s\n", errDo)
		default:
			fmt.Fprintln(os.Stderr, "Done")
			if err := db.ReplayBrigade(false, false, false, true, true); err != nil {
				log.Fatalf("replay brigade: %s", err)
			}
		}

		return
	}

	switch {
	case errors.Is(errDo, ErrNeedRestart):
		fmt.Fprintln(os.Stderr, "Need restart")
		os.Exit(2)
	case errors.Is(errDo, ErrNeedFullReplay):
		fmt.Fprintln(os.Stderr, "Need full replay")
		os.Exit(3)
	case errDo != nil:
		log.Fatalf("Can't do: %s\n", errDo)
	default:
		fmt.Fprintln(os.Stderr, "Done")
		os.Exit(0)
	}
}

func parseArgs() (string, string, string, netip.AddrPort, bool, bool, error) {
	var (
		id       string
		dbdir    string
		addrPort netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return "", "", "", addrPort, false, false, fmt.Errorf("cannot define user: %w", err)
	}

	dryRun := flag.Bool("n", false, "Dry run")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")
	replay := flag.Bool("r", false, "replay changes (default: false)")

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return "", "", "", addrPort, false, false, fmt.Errorf("addr: %w", err)
		}
	}

	flag.Parse()

	if flag.NArg() != 1 {
		return "", "", "", addrPort, false, false, ErrInvalidArgs
	}

	filename := flag.Arg(0)

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return "", "", "", addrPort, false, false, fmt.Errorf("dbdir dir: %w", err)
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

	return filename, id, dbdir, addrPort, *dryRun, *replay, nil
}

// Do - do replay.
func Do(db *storage.BrigadeStorage, newfile string, dryRun bool) error {
	fresh, errPatch := readNewFile(newfile, db.BrigadeID)
	if errPatch != nil {
		return fmt.Errorf("read new file: %w", errPatch)
	}

	f, old, errPatch := db.OpenDbToModify()
	if errPatch != nil {
		return fmt.Errorf("open db: %w", errPatch)
	}

	defer f.Close()

	newdata, errPatch := patchBrigadeCommon(fresh, old)
	if errPatch != nil && !errors.Is(errPatch, ErrNeedFullReplay) {
		return fmt.Errorf("patch common: %w", errPatch)
	}

	switch dryRun {
	case true:
		fmt.Fprintf(os.Stderr, "Brigade %s: dry run\n", fresh.BrigadeID)
	default:
		fmt.Fprintf(os.Stderr, "Brigade %s: apply patch\n", fresh.BrigadeID)

		if err := f.Commit(newdata); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	if old.Mode != fresh.Mode {
		return ErrNeedRestart
	}

	return errPatch
}

func readNewFile(filename, brigadeID string) (*storage.Brigade, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", filename, err)
	}

	defer file.Close()

	data := &storage.Brigade{}

	if err := json.NewDecoder(file).Decode(data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if data.BrigadeID != brigadeID {
		return nil, fmt.Errorf("check: %w", storage.ErrUnknownBrigade)
	}

	if data.Mode == "" {
		data.Mode = storage.ModeBrigade
	}

	if data.CloakFakeDomain == "" && data.CloakFaekDomain != "" {
		data.CloakFakeDomain = data.CloakFaekDomain
	}

	return data, nil
}

func patchBrigadeCommon(fresh, old *storage.Brigade) (*storage.Brigade, error) {
	newdata := *fresh

	newdata.BrigadeCounters = old.BrigadeCounters
	newdata.StatsCountersStack = old.StatsCountersStack
	newdata.Endpoints = old.Endpoints

	newdata.Users = make([]*storage.User, 0, len(fresh.Users))

	deletedUsers(&newdata, old, fresh)
	freshUsers(&newdata, old, fresh)

	sort.Slice(newdata.Users, func(i, j int) bool {
		return newdata.Users[i].IsBrigadier || !newdata.Users[j].IsBrigadier && (newdata.Users[i].UserID.String() > newdata.Users[j].UserID.String())
	})

	if string(old.WgPublicKey) != string(fresh.WgPublicKey) ||
		string(old.WgPrivateShufflerEnc) != string(fresh.WgPrivateShufflerEnc) ||

		old.CloakFakeDomain != fresh.CloakFakeDomain ||

		old.OvCAKeyShufflerEnc != fresh.OvCAKeyShufflerEnc ||
		old.OvCACertPemGzipBase64 != fresh.OvCACertPemGzipBase64 ||

		old.IPSecPSK != fresh.IPSecPSK ||
		old.IPSecPSKShufflerEnc != fresh.IPSecPSKShufflerEnc ||

		old.EndpointIPv4 != fresh.EndpointIPv4 ||
		old.EndpointDomain != fresh.EndpointDomain ||
		old.EndpointPort != fresh.EndpointPort ||

		old.OutlinePort != fresh.OutlinePort ||

		old.Proto0FakeDomain != fresh.Proto0FakeDomain ||
		old.Proto0Port != fresh.Proto0Port ||

		old.KeydeskIPv6 != fresh.KeydeskIPv6 ||

		old.IPv4CGNAT != fresh.IPv4CGNAT ||
		old.IPv6ULA != fresh.IPv6ULA {
		fmt.Fprintf(os.Stderr, "Brigade %s: need full replay\n", fresh.BrigadeID)

		return &newdata, ErrNeedFullReplay
	}

	return &newdata, nil
}

func deletedUsers(newdata, old, fresh *storage.Brigade) {
	// only add users that are must be deleted.
OLD:
	for _, oldUser := range old.Users {
		for _, freshUser := range fresh.Users {
			if freshUser.DelayedDeletion {
				continue
			}

			if isUsersEqual(freshUser, oldUser) {
				continue OLD
			}
		}

		fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) absent\n", newdata.BrigadeID, oldUser.UserID, oldUser.Name)

		// if blocked, just skip
		if !oldUser.IsBlocked {
			oldUser.DelayedDeletion = true
			oldUser.DelayedCreation = false
			oldUser.DelayedReplay = false
			oldUser.DelayedBlocking = false
			newdata.Users = append(newdata.Users, oldUser)

			fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) marked for deletion\n", newdata.BrigadeID, oldUser.UserID, oldUser.Name)
		}
	}
}

func freshUsers(newdata, old, fresh *storage.Brigade) {
UPD:
	for _, freshUser := range fresh.Users {
		if freshUser.DelayedDeletion {
			continue
		}

		freshUser.DelayedCreation = false
		freshUser.DelayedDeletion = false
		freshUser.DelayedReplay = false
		freshUser.DelayedBlocking = false

		for _, oldUser := range old.Users {
			if isUsersEqual(oldUser, freshUser) {
				// fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) finded\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)

				if isUserModified(freshUser, oldUser) {
					fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) modified\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)
					if !freshUser.IsBlocked && !oldUser.IsBlocked {
						fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) marked for replay\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)
						freshUser.DelayedReplay = true
					}
				}

				switch {
				case oldUser.IsBlocked && !freshUser.IsBlocked:
					fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) marked for creation\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)
					freshUser.DelayedCreation = true
				case !oldUser.IsBlocked && freshUser.IsBlocked:
					fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) marked for blocking\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)
					freshUser.DelayedBlocking = true
				}

				newdata.Users = append(newdata.Users, freshUser)

				continue UPD
			}
		}

		fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) new user\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)

		// add new user
		if !freshUser.IsBlocked {
			freshUser.DelayedCreation = true
			fmt.Fprintf(os.Stderr, "Brigade %s: user %s (%s) marked for creation\n", newdata.BrigadeID, freshUser.UserID, freshUser.Name)
		}

		newdata.Users = append(newdata.Users, freshUser)
	}
}

func isUsersEqual(newuser, user *storage.User) bool {
	if newuser.UserID != user.UserID ||
		string(newuser.WgPublicKey) != string(user.WgPublicKey) {
		return false
	}

	return true
}

func isUserModified(newuser, user *storage.User) bool {
	switch {
	case user.IsBrigadier != newuser.IsBrigadier:
		fmt.Fprintf(os.Stderr, "   brigadier changed\n")
		return true
	case user.IPv4Addr != newuser.IPv4Addr:
		fmt.Fprintf(os.Stderr, "    IPv4 changed (%s) -> (%s)\n", user.IPv4Addr, newuser.IPv4Addr)
		return true
	case user.IPv6Addr != newuser.IPv6Addr:
		fmt.Fprintf(os.Stderr, "    IPv6 changed (%s) -> (%s)\n", user.IPv6Addr, newuser.IPv6Addr)
		return true
	case user.EndpointDomain != newuser.EndpointDomain:
		fmt.Fprintf(os.Stderr, "    EndpointDomain changed (%s) -> (%s)\n", user.EndpointDomain, newuser.EndpointDomain)
		return true
	case string(user.WgPSKShufflerEnc) != string(newuser.WgPSKShufflerEnc):
		fmt.Fprintf(os.Stderr, "    WgPSKShufflerEnc changed (%s) -> (%s)\n", user.WgPSKShufflerEnc, newuser.WgPSKShufflerEnc)
		return true
	case user.CloakByPassUIDShufflerEnc != newuser.CloakByPassUIDShufflerEnc:
		fmt.Fprintf(os.Stderr, "    CloakByPassUIDShufflerEnc changed (%s) -> (%s)\n", user.CloakByPassUIDShufflerEnc, newuser.CloakByPassUIDShufflerEnc)
		return true
	case user.OvCSRGzipBase64 != newuser.OvCSRGzipBase64:
		fmt.Fprintf(os.Stderr, "    OvCSRGzipBase64 changed (%s) -> (%s)\n", user.OvCSRGzipBase64, newuser.OvCSRGzipBase64)
		return true
	case user.IPSecUsernameShufflerEnc != newuser.IPSecUsernameShufflerEnc:
		fmt.Fprintf(os.Stderr, "    IPSecUsernameShufflerEnc changed (%s) -> (%s)\n", user.IPSecUsernameShufflerEnc, newuser.IPSecUsernameShufflerEnc)
		return true
	case user.IPSecPasswordShufflerEnc != newuser.IPSecPasswordShufflerEnc:
		fmt.Fprintf(os.Stderr, "    IPSecPasswordShufflerEnc changed (%s) -> (%s)\n", user.IPSecPasswordShufflerEnc, newuser.IPSecPasswordShufflerEnc)
		return true
	case user.OutlineSecretShufflerEnc != newuser.OutlineSecretShufflerEnc:
		fmt.Fprintf(os.Stderr, "    OutlineSecretShufflerEnc changed (%s) -> (%s)\n", user.OutlineSecretShufflerEnc, newuser.OutlineSecretShufflerEnc)
		return true
	case user.Proto0SecretShufflerEnc != newuser.Proto0SecretShufflerEnc:
		fmt.Fprintf(os.Stderr, "    Proto0SecretShufflerEnc changed (%s) -> (%s)\n", user.Proto0SecretShufflerEnc, newuser.Proto0SecretShufflerEnc)
		return true
	}

	/*if user.IsBrigadier != newuser.IsBrigadier ||
		user.IPv4Addr != newuser.IPv4Addr ||
		user.IPv6Addr != newuser.IPv6Addr ||
		user.EndpointDomain != newuser.EndpointDomain ||
		string(user.WgPSKRouterEnc) != string(newuser.WgPSKRouterEnc) ||
		string(user.WgPSKShufflerEnc) != string(newuser.WgPSKShufflerEnc) ||

		user.CloakByPassUIDRouterEnc != newuser.CloakByPassUIDRouterEnc ||
		user.CloakByPassUIDShufflerEnc != newuser.CloakByPassUIDShufflerEnc ||
		user.OvCSRGzipBase64 != newuser.OvCSRGzipBase64 ||

		user.IPSecUsernameRouterEnc != newuser.IPSecUsernameRouterEnc ||
		user.IPSecUsernameShufflerEnc != newuser.IPSecUsernameShufflerEnc ||
		user.IPSecPasswordRouterEnc != newuser.IPSecPasswordRouterEnc ||
		user.IPSecPasswordShufflerEnc != newuser.IPSecPasswordShufflerEnc ||

		user.OutlineSecretRouterEnc != newuser.OutlineSecretRouterEnc ||
		user.OutlineSecretShufflerEnc != newuser.OutlineSecretShufflerEnc ||
		user.Proto0SecretRouterEnc != newuser.Proto0SecretRouterEnc ||
		user.Proto0SecretShufflerEnc != newuser.Proto0SecretShufflerEnc {
		return true
	}*/

	return false
}
