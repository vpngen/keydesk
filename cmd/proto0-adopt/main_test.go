package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"golang.org/x/crypto/nacl/box"
)

type keys struct {
	rPub, rPriv, sPub, sPriv *[32]byte
}

func newKeys(t *testing.T) keys {
	t.Helper()

	rPub, rPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	sPub, sPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return keys{rPub, rPriv, sPub, sPriv}
}

func wgKey(t *testing.T) []byte {
	t.Helper()

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	return b
}

func open(t *testing.T, enc string, pub, priv *[32]byte) string {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("base64: %s", err)
	}

	out, ok := box.OpenAnonymous(nil, raw, pub, priv)
	if !ok {
		t.Fatal("cannot open sealed secret")
	}

	return string(out)
}

func TestAdopt(t *testing.T) {
	k := newKeys(t)

	had, missing, urlsafe, noclient := wgKey(t), wgKey(t), wgKey(t), wgKey(t)
	idMissing, idURL, idHad := uuid.NewString(), uuid.NewString(), uuid.NewString()

	data := &storage.Brigade{Users: []*storage.User{
		{UserID: uuid.New(), Name: "had", WgPublicKey: had, Proto0SecretRouterEnc: "KEEP-R", Proto0SecretShufflerEnc: "KEEP-S"},
		{UserID: uuid.New(), Name: "missing", WgPublicKey: missing},
		{UserID: uuid.New(), Name: "urlsafe", WgPublicKey: urlsafe},
		{UserID: uuid.New(), Name: "noclient", WgPublicKey: noclient},
	}}

	live := &Live{Clients: []Client{
		{ID: idHad, Email: base64.StdEncoding.EncodeToString(had), ShortID: "aaaaaaaaaaaa"},
		{ID: idMissing, Email: base64.StdEncoding.EncodeToString(missing), ShortID: "0123456789ab"},
		{ID: idURL, Email: base64.RawURLEncoding.EncodeToString(urlsafe), ShortID: "BCDEF0123456"},
		{ID: uuid.NewString(), Email: base64.StdEncoding.EncodeToString(wgKey(t)), ShortID: "cccccccccccc"},
		{ID: uuid.NewString(), Email: "not-a-key", ShortID: "dddddddddddd"},
		{ID: uuid.NewString(), Email: base64.StdEncoding.EncodeToString(wgKey(t)), ShortID: "xyz"},
	}}

	rep, err := adopt(data, live, k.rPub, k.sPub)
	if err != nil {
		t.Fatal(err)
	}

	if rep.Users != 4 || rep.HadSecret != 1 || len(rep.Adopted) != 2 || rep.NoClient != 1 || rep.UnusedClient != 1 || len(rep.BadClients) != 2 {
		t.Fatalf("report: %+v", rep)
	}

	if data.Users[0].Proto0SecretRouterEnc != "KEEP-R" || data.Users[0].Proto0SecretShufflerEnc != "KEEP-S" {
		t.Fatal("a user that already had a secret was changed")
	}

	if data.Users[3].Proto0SecretRouterEnc != "" || data.Users[3].Proto0SecretShufflerEnc != "" {
		t.Fatal("a user without a live client got a secret")
	}

	for _, c := range []struct {
		u  *storage.User
		id string
		sh string
	}{
		{data.Users[1], idMissing, "0123456789ab"},
		{data.Users[2], idURL, "bcdef0123456"},
	} {
		want := c.sh + "-" + strings.ReplaceAll(c.id, "-", "")

		if got := open(t, c.u.Proto0SecretRouterEnc, k.rPub, k.rPriv); got != want {
			t.Fatalf("router secret of %s: got %q, want %q", c.u.Name, got, want)
		}

		if got := open(t, c.u.Proto0SecretShufflerEnc, k.sPub, k.sPriv); got != want {
			t.Fatalf("shuffler secret of %s: got %q, want %q", c.u.Name, got, want)
		}
	}
}

func TestAdoptDuplicateClient(t *testing.T) {
	k := newKeys(t)
	key := wgKey(t)

	live := &Live{Clients: []Client{
		{ID: uuid.NewString(), Email: base64.StdEncoding.EncodeToString(key), ShortID: "0123456789ab"},
		{ID: uuid.NewString(), Email: base64.RawURLEncoding.EncodeToString(key), ShortID: "0123456789ac"},
	}}

	if _, err := adopt(&storage.Brigade{}, live, k.rPub, k.sPub); err == nil {
		t.Fatal("two clients for one key must be refused")
	}
}

// The whole file path: the lock, the commit, the backup, and that a JSON round trip
// through storage.Brigade keeps the secrets.
func TestDoWritesFileAndBackup(t *testing.T) {
	k := newKeys(t)
	dir := t.TempDir()
	key := wgKey(t)
	id := uuid.NewString()

	brigade := storage.Brigade{
		BrigadeID: "TESTBRIGADE",
		Users:     []*storage.User{{UserID: uuid.New(), Name: "u", WgPublicKey: key}},
	}

	raw, err := json.Marshal(brigade)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, storage.BrigadeFilename), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       "TESTBRIGADE",
		BrigadeFilename: filepath.Join(dir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dir, storage.BrigadeSpinlockFilename),
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}

	live := &Live{Clients: []Client{{ID: id, Email: base64.StdEncoding.EncodeToString(key), ShortID: "0123456789ab"}}}

	// a dry run writes nothing
	if err := Do(db, live, k.rPub, k.sPub, true); err != nil {
		t.Fatal(err)
	}

	if after, _ := os.ReadFile(db.BrigadeFilename); string(after) != string(raw) {
		t.Fatal("dry run changed brigade.json")
	}

	if err := Do(db, live, k.rPub, k.sPub, false); err != nil {
		t.Fatal(err)
	}

	backups, _ := filepath.Glob(db.BrigadeFilename + ".bak-proto0-adopt-*")
	if len(backups) != 1 {
		t.Fatalf("want one backup, got %v", backups)
	}

	if b, _ := os.ReadFile(backups[0]); string(b) != string(raw) {
		t.Fatal("backup is not the original file")
	}

	var got storage.Brigade
	b, _ := os.ReadFile(db.BrigadeFilename)
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}

	want := "0123456789ab-" + strings.ReplaceAll(id, "-", "")
	if s := open(t, got.Users[0].Proto0SecretRouterEnc, k.rPub, k.rPriv); s != want {
		t.Fatalf("committed router secret: got %q, want %q", s, want)
	}

	// a second run finds nothing to do and makes no second backup
	if err := Do(db, live, k.rPub, k.sPub, false); err != nil {
		t.Fatal(err)
	}

	if backups, _ := filepath.Glob(db.BrigadeFilename + ".bak-proto0-adopt-*"); len(backups) != 1 {
		t.Fatalf("second run made a backup: %v", backups)
	}
}
