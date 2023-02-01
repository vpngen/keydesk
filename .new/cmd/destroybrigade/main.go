package main

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http/httputil"
	"net/netip"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vpngen/keydesk/env"
	"github.com/vpngen/keydesk/user"
)

/*
Remove brigade
  * Delete brigade
Database manipulations
  * Remove privileges
  * Remove database schema
  * Remove schema role
This programm need environments:
  * database name: content of /etc/keydesk/dbname
  * router pub key: content of /etc/keydesk/router.pub
  * shuffler pub key: content of /etc/keydesk/shuffler.pub
*/

const vpngineUser = "qrc"

const etcDefaultPath = "/etc/keydesk"

const (
	sqlDropSchema        = "DROP SCHEMA %s CASCADE"
	sqlDropRole          = "DROP ROLE %s"
	sqlNotifyDelBrigade  = "SELECT pg_notify('vpngen', $1)"
	sqlRevokeRoleTables  = "REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA %s FROM %s"
	sqlSelectBrigadeInfo = "SELECT wg_private_router,endpoint_ipv4 FROM %s FOR UPDATE"
)

func parseArgs() (bool, string, error) {
	brigadeID := flag.String("id", "", "brigadier_id")
	chunked := flag.Bool("ch", false, "chunked output")

	flag.Parse()

	// brigadeID must be base32 decodable.
	id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(*brigadeID)
	if err != nil {
		return false, "", fmt.Errorf("id base32: %s: %w", *brigadeID, err)
	}

	_, err = uuid.FromBytes(id)
	if err != nil {
		return false, "", fmt.Errorf("id uuid: %s: %w", *brigadeID, err)
	}

	return *chunked, *brigadeID, nil
}

func do() error {
	ctx := context.Background()

	tx, err := env.Env.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	schema := (pgx.Identifier{env.Env.BrigadierID}).Sanitize()

	var (
		wg_private_router []byte
		endpoint_ipv4     netip.Addr
	)

	rows, err := tx.Query(ctx, fmt.Sprintf(sqlSelectBrigadeInfo, (pgx.Identifier{env.Env.BrigadierID, "brigade"}).Sanitize()))
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("brigadier query: %w", err)
	}

	_, err = pgx.ForEachRow(rows, []any{&wg_private_router, &endpoint_ipv4}, func() error {
		//fmt.Printf("Brigade:\nwg_public: %v\nendpoint_ipv4: %v\ndns_ipv4: %v\ndns_ipv6: %v\nipv4_cgnat: %v\nipv6_ula: %v\n", wg_public, endpoint_ipv4, dns_ipv4, dns_ipv6, ipv4_cgnat, ipv6_ula)

		return nil
	})
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("brigadier row: %w", err)
	}

	brigadeNotify := &user.SrvNotify{
		T:        user.NotifyDelBrigade,
		Endpoint: user.NewEndpoint(endpoint_ipv4),
		Brigade: user.SrvBrigade{
			ID:          env.Env.BrigadierID,
			IdentityKey: wg_private_router,
		},
	}

	notify, err := json.Marshal(brigadeNotify)
	if err != nil {
		tx.Rollback(context.Background())

		return fmt.Errorf("marshal notify: %w", err)
	}

	_, err = tx.Exec(ctx, sqlNotifyDelBrigade, notify)
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("notify: %w", err)
	}

	// revoke priv from qrc
	_, err = tx.Exec(ctx, fmt.Sprintf(sqlRevokeRoleTables, schema, vpngineUser))
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("revoke on tables to qrc: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlDropSchema, schema))
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("drop schema: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlDropRole, schema))
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("drop role: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func main() {
	var w io.WriteCloser

	confDir := os.Getenv("CONFDIR")
	if confDir == "" {
		confDir = etcDefaultPath
	}

	err := env.ReadConfigs(confDir)
	if err != nil {
		log.Fatalf("Can't read config files: %s", err)
	}

	chunked, brigadeID, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	env.Env.BrigadierID = brigadeID

	err = env.CreatePool()
	if err != nil {
		log.Fatalln(err)
	}

	defer env.Env.DB.Close()

	// just do it.
	err = do()
	if err != nil {
		log.Fatalf("Can't destroy brigade: %s", err)
	}

	switch chunked {
	case true:
		w = httputil.NewChunkedWriter(os.Stdout)
		defer w.Close()
	default:
		w = os.Stdout
	}

	_, err = w.Write([]byte{})
	if err != nil {
		log.Fatalf("Can't write: %s", err)
	}

}
