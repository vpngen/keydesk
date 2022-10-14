package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	ctx := context.Background()

	config, err := pgxpool.ParseConfig("host=/var/run/postgresql dbname=castle")
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		panic(err)
	}

	tx.Conn().TypeMap().RegisterType(&pgtype.Type{Name: "wireguard_key", OID: pgtype.ByteaOID, Codec: pgtype.ByteaCodec{}})
	tx.Conn().TypeMap().RegisterType(&pgtype.Type{Name: "sealed_wireguard_key", OID: pgtype.ByteaOID, Codec: pgtype.ByteaCodec{}})

	userID := uuid.New()

	_, err = tx.Exec(ctx,
		`INSERT INTO "PWO2KLP4XJDLFNL6YQVDHTDZEM".users (user_id, user_callsign, is_brigadier, wg_public, wg_psk_router, wg_psk_shuffler, user_ipv4, user_ipv6) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID.String(), "Вася", true, []byte{0, 1, 2}, []byte{3, 4, 5}, []byte{6, 7, 8}, "100.65.2.33", "fd01::15",
	)
	if err != nil {
		tx.Rollback(ctx)

		panic(err)
	}

	tx.Commit(ctx)

	/*_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS "PWO2KLP4XJDLFNL6YQVDHTDZEM".test_bytea`)

	if err != nil {
		panic(err)
	}
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS "PWO2KLP4XJDLFNL6YQVDHTDZEM".test_bytea (
		id serial PRIMARY KEY,
		user_id UUID
	)`)

	if err != nil {
		panic(err)
	}

	userID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO "PWO2KLP4XJDLFNL6YQVDHTDZEM".test_bytea (user_id) VALUES ($1)`, userID.String())
	if err != nil {
		panic(err)
	}

	// var id int

	// err = row.Scan(&id)
	// if err != nil {
	// 	panic(err)
	// }

	row := pool.QueryRow(ctx, `SELECT user_id from "PWO2KLP4XJDLFNL6YQVDHTDZEM".test_bytea WHERE id = $1`, 1)

	var res []byte

	row.Scan(&res)
	if err != nil {
		panic(err)
	}

	userID1, _ := uuid.FromBytes(res)
	fmt.Printf("in=%q  out=%q\n", userID, userID1)*/
}
