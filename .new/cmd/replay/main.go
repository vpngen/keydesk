package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"

	"github.com/jackc/pgx/v5"

	"github.com/vpngen/keydesk/env"
	"github.com/vpngen/keydesk/user"
)

const etcDefaultPath = "/etc/keydesk"

const (
	sqlGrantRoleTables = "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA %s TO %s"
	sqlNotify          = "SELECT pg_notify('vpngen', $1)"
)

var (
	ErrInvalidArgs = errors.New("invalid arguments")
)

func parseArgs() (bool, bool, bool, error) {
	bonly := flag.Bool("b", false, "brigades only")
	uonly := flag.Bool("u", false, "users only")
	clean := flag.Bool("c", false, "clean setup (with deletion)")

	flag.Parse()

	if (*bonly && *uonly) || (*clean && *uonly) {
		return false, false, false, ErrInvalidArgs
	}

	return *clean, *bonly, *uonly, nil
}

func brigadeList(ctx context.Context, tx pgx.Tx) ([]string, error) {
	var (
		bid      string
		brigades []string
	)

	sqlListBrigades := `
	SELECT n.nspname 
		FROM pg_catalog.pg_namespace n 
		WHERE 
			n.nspname !~ '^pg_' 
		AND 
			n.nspname <> '_v' 
		AND 
			n.nspname <> 'meta' 
		AND 
			n.nspname <> 'public' 
		AND 
			n.nspname <> 'information_schema'`

	rows, err := tx.Query(ctx, sqlListBrigades)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	_, err = pgx.ForEachRow(rows, []any{&bid}, func() error {
		brigades = append(brigades, bid)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("row: %w", err)
	}

	return brigades, nil
}

func notifyDeleteBrigade(ctx context.Context, tx pgx.Tx, brigadeID string, endpoint_ipv4 netip.Addr, wg_private_router []byte) error {
	brigadeNotify := &user.SrvNotify{
		T:        user.NotifyDelBrigade,
		Endpoint: user.NewEndpoint(endpoint_ipv4),
		Brigade: user.SrvBrigade{
			ID:          brigadeID,
			IdentityKey: wg_private_router,
		},
	}

	notify, err := json.Marshal(brigadeNotify)
	if err != nil {
		return fmt.Errorf("marshal notify: %w", err)
	}

	_, err = tx.Exec(ctx, sqlNotify, notify)
	if err != nil {
		return fmt.Errorf("notify: %w", err)
	}

	return nil
}

func notifyCreateBrigade(ctx context.Context, tx pgx.Tx, brigadeID string, endpoint_ipv4 netip.Addr, wg_private_router []byte) error {
	brigadeNotify := &user.SrvNotify{
		T:        user.NotifyNewBrigade,
		Endpoint: user.NewEndpoint(endpoint_ipv4),
		Brigade: user.SrvBrigade{
			ID:          brigadeID,
			IdentityKey: wg_private_router,
		},
	}

	notify, err := json.Marshal(brigadeNotify)
	if err != nil {
		return fmt.Errorf("marshal notify: %w", err)
	}

	_, err = tx.Exec(ctx, sqlNotify, notify)
	if err != nil {
		return fmt.Errorf("notify: %w", err)
	}

	return nil
}

func notifyCreateUser(ctx context.Context, tx pgx.Tx, brigadeID string, endpoint_ipv4 netip.Addr, wg_public []byte, userID string, user_wg_public []byte, is_brigadier bool) error {
	userNotify := &user.SrvNotify{
		T:        user.NotifyNewUser,
		Endpoint: user.NewEndpoint(endpoint_ipv4),
		Brigade: user.SrvBrigade{
			ID:          brigadeID,
			IdentityKey: wg_public,
		},
		User: user.SrvUser{
			ID:          userID,
			WgPublicKey: user_wg_public,
			IsBrigadier: is_brigadier,
		},
	}

	notify, err := json.Marshal(userNotify)
	if err != nil {
		return fmt.Errorf("marshal notify: %w", err)
	}

	fmt.Fprintf(os.Stderr, "notify json: %s\n", string(notify))

	_, err = tx.Exec(ctx, sqlNotify, notify)
	if err != nil {
		return fmt.Errorf("notify: %w", err)
	}

	return nil
}

func userList(ctx context.Context, tx pgx.Tx, bid string) ([]string, error) {
	var (
		users []string
		uid   string
	)

	sqlListUsers := "SELECT user_id FROM %s ORDER BY is_brigadier"
	rows, err := tx.Query(ctx,
		fmt.Sprintf(sqlListUsers,
			(pgx.Identifier{bid, "users"}).Sanitize(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	_, err = pgx.ForEachRow(rows, []any{&uid}, func() error {
		users = append(users, uid)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("row: %w", err)
	}

	return users, nil
}

func do(clean, bonly, uonly bool) error {
	ctx := context.Background()

	tx, err := env.Env.DB.Begin(ctx)
	if err != nil {
		tx.Rollback(ctx)

		return fmt.Errorf("begin: %w", err)
	}

	brigades, err := brigadeList(ctx, tx)
	if err != nil {
		return fmt.Errorf("brigades list: %w", err)
	}

	for _, brigadeID := range brigades {
		var (
			wg_private_router []byte
			wg_public         []byte
			endpoint_ipv4     netip.Addr
		)

		sqlSelectBrigadeInfo := "SELECT wg_public,wg_private_router,endpoint_ipv4 FROM %s LIMIT 1"
		err := tx.QueryRow(ctx,
			fmt.Sprintf(sqlSelectBrigadeInfo,
				(pgx.Identifier{brigadeID, "brigade"}).Sanitize(),
			),
		).Scan(
			&wg_public,
			&wg_private_router,
			&endpoint_ipv4,
		)
		if err != nil {
			tx.Rollback(ctx)

			return fmt.Errorf("brigadier query: %w", err)
		}

		if !uonly {
			if clean {
				fmt.Fprintf(os.Stderr, "Notify: destroy brigade: %s\n", brigadeID)
				err := notifyDeleteBrigade(ctx, tx, brigadeID, endpoint_ipv4, wg_private_router)
				if err != nil {
					tx.Rollback(ctx)

					return fmt.Errorf("brigade notify: %w", err)
				}
			}

			fmt.Fprintf(os.Stderr, "Notify: create brigade: %s\n", brigadeID)
			err := notifyCreateBrigade(ctx, tx, brigadeID, endpoint_ipv4, wg_private_router)
			if err != nil {
				tx.Rollback(ctx)

				return fmt.Errorf("brigade notify: %w", err)
			}
		}

		users, err := userList(ctx, tx, brigadeID)
		if err != nil {
			return fmt.Errorf("users list: %w", err)
		}
		for _, userID := range users {
			var (
				user_wg_public []byte
				is_brigadier   bool
			)

			sqlSelectUserInfo := "SELECT wg_public,is_brigadier FROM %s WHERE user_id=$1"
			err := tx.QueryRow(ctx,
				fmt.Sprintf(sqlSelectUserInfo,
					(pgx.Identifier{brigadeID, "users"}).Sanitize(),
				),
				userID,
			).Scan(
				&user_wg_public,
				&is_brigadier,
			)
			if err != nil {
				tx.Rollback(ctx)

				return fmt.Errorf("user query: %w", err)
			}

			if !bonly || is_brigadier {
				fmt.Fprintf(os.Stderr, "Notify: create user: %s\n", userID)
				err := notifyCreateUser(ctx, tx, brigadeID, endpoint_ipv4, wg_public, userID, user_wg_public, is_brigadier)
				if err != nil {
					tx.Rollback(ctx)

					return fmt.Errorf("user notify: %w", err)
				}
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func main() {
	confDir := os.Getenv("CONFDIR")
	if confDir == "" {
		confDir = etcDefaultPath
	}

	err := env.ReadConfigs(confDir)
	if err != nil {
		log.Fatalf("Can't read config files: %s", err)
	}

	clean, bonly, uonly, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	err = env.CreatePool()
	if err != nil {
		log.Fatalln(err)
	}

	defer env.Env.DB.Close()

	err = do(clean, bonly, uonly)
	if err != nil {
		log.Fatalln(err)
	}
}
