package env

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vpngen/vpngine/naclkey"
)

const (
	dbnameFilename            = "dbname"
	routerPublicKeyFilename   = "router.pub"
	shufflerPublicKeyFilename = "shuffler.pub"
	etcDefaultPath            = "/etc/keydesk"
)

const maxPostgresqlNameLen = 63

const postgresqlSocket = "/var/run/postgresql"

// KeydeskEnv - struct type for shared vars.
type KeydeskEnv struct {
	BrigadierID                        string
	DBName                             string
	RouterPublicKey, ShufflerPublicKey [naclkey.NaclBoxKeyLength]byte
	DB                                 *pgxpool.Pool
}

// Env - shared vars.
var Env KeydeskEnv

func ReadConfigs(path string) error {
	f, err := os.Open(filepath.Join(path, dbnameFilename))
	if err != nil {
		return fmt.Errorf("can't open: %s: %w", dbnameFilename, err)
	}

	dbname, err := io.ReadAll(io.LimitReader(f, maxPostgresqlNameLen))
	if err != nil {
		return fmt.Errorf("can't read: %s: %w", dbnameFilename, err)
	}

	routerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, routerPublicKeyFilename))
	if err != nil {
		return fmt.Errorf("router key: %w", err)
	}

	shufflerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, shufflerPublicKeyFilename))
	if err != nil {
		return fmt.Errorf("shuffler key: %w", err)
	}

	Env.DBName = string(bytes.Trim(dbname, "\r\n "))
	Env.RouterPublicKey = routerPublicKey
	Env.ShufflerPublicKey = shufflerPublicKey
	return nil
}

func CreatePool() error {
	//config, err := pgxpool.ParseConfig(fmt.Sprintf("host=%s user=%s dbname=%s", postgresqlSocket, Env.BrigadierID, Env.DBName))
	config, err := pgxpool.ParseConfig(fmt.Sprintf("host=%s dbname=%s", postgresqlSocket, Env.DBName))
	if err != nil {
		return fmt.Errorf("Can't parse conn string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("Can't create pool: %w", err)
	}

	Env.DB = pool

	return nil
}
