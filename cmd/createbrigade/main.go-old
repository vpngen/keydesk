package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/nacl/box"

	"github.com/vpngen/keydesk/env"
	"github.com/vpngen/keydesk/user"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

/*
Database manipulations
  * Create database schema
  * !!Create schema role
  * Create endpoint credentials
Create brigadier
  * Create brigadier record
  * Create special brigadier wg-user, get brigadier IPv6 and wg.conf
Create config
  * Database role credentials
  * Brigadier IPv6-address (p2p brigadier2keykdesk)

This programm need environments:
  * database name: content of /etc/keydesk/dbname
  * router pub key: content of /etc/keydesk/router.pub
  * shuffler pub key: content of /etc/keydesk/shuffler.pub
*/

const vpngineUser = "qrc"

const etcDefaultPath = "/etc/keydesk"

const (
	sqlCreateSchema       = "CREATE SCHEMA %s"
	sqlCreateRole         = "CREATE ROLE %s LOGIN"
	sqlTrimRole           = "REVOKE ALL ON ALL TABLES IN SCHEMA  pg_catalog FROM public, %s"
	sqlGrantRoleSchema    = "GRANT USAGE ON SCHEMA %s TO %s"
	sqlCreateTableBrigade = "CREATE TABLE %s (LIKE meta.tpl_brigade INCLUDING ALL)"
	sqlCreateTableUsers   = "CREATE TABLE %s (LIKE meta.tpl_users INCLUDING ALL)"
	sqlCreateTableQuota   = "CREATE TABLE %s (LIKE meta.tpl_quota INCLUDING ALL)"
	sqlCreateTableKeydesk = "CREATE TABLE %s (LIKE meta.tpl_keydesk INCLUDING ALL)"
	sqlAlterTableQuota    = "ALTER TABLE %s ADD CONSTRAINT quota_user_id_fkey FOREIGN KEY (user_id) REFERENCES %s(user_id) ON DELETE CASCADE"
	sqlCreateTablePersons = "CREATE TABLE %s (LIKE meta.tpl_persons INCLUDING ALL)"
	sqlAlterTablePersons  = "ALTER TABLE %s ADD CONSTRAINT person_user_id_fkey FOREIGN KEY (user_id) REFERENCES %s(user_id) ON DELETE CASCADE"
	sqlInsertBrigade      = `INSERT INTO %s
	(	wg_public, 
		wg_private_router, 
		wg_private_shuffler, 
		endpoint_ipv4, 
		dns_ipv4, 
		dns_ipv6, 
		keydesk_ipv6, 
		ipv4_cgnat, 
		ipv6_ula ) 
	VALUES 
	(	$1, -- 'wg_public'
		$2, -- 'wg_private_router'
		$3, -- 'wg_private_shuffler
		$4, -- 'endpoint_ipv4' 
		$5, -- 'dns_ipv4',
		$6, -- 'dns_ipv6' 
		$7, -- 'keydesk_ipv6' 
		$8, -- 'ipv4_cgnat' 
		$9 -- 'ipv6_ula 
	)`
	sqlInsertKeydesk    = `INSERT INTO %s DEFAULT VALUES ON CONFLICT DO NOTHING`
	sqlGrantRoleTables  = "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA %s TO %s"
	sqlNotifyNewBrigade = "SELECT pg_notify('vpngen', $1)"
)

const (
	CGNATPrefix = "100.64.0.0/10"
	ULAPrefix   = "fd00::/8"
)

type tOpts struct {
	brigadeID     string
	brigadierName string
	endpointIPv4  netip.Addr
	dnsIPv4       netip.Addr
	dnsIPv6       netip.Addr
	IPv4CGNAT     netip.Prefix
	IPv6ULA       netip.Prefix
	keydeskIPv6   netip.Addr
	person        namesgenerator.Person
}

// Args errors.
var (
	ErrInvalidEndpointIPv4  = errors.New("invalid ip4 endpoint")
	ErrInvalidDNS4          = errors.New("invalid dns ip4 endpoint")
	ErrInvalidDNS6          = errors.New("invalid dns ip6 endpoint")
	ErrInvalidNetCGNAT      = errors.New("invalid cgnat net")
	ErrInvalidNetULA        = errors.New("invalid ula net")
	ErrInvalidKeydeskIPv6   = errors.New("invalid keydesk ip6 endpoint")
	ErrEmptyBrigadierName   = errors.New("empty brigadier name")
	ErrInvalidBrigadierName = errors.New("invalid brigadier name")
	ErrEmptyPersonName      = errors.New("empty person name")
	ErrEmptyPersonDesc      = errors.New("empty person desc")
	ErrEmptyPersonURL       = errors.New("empty person url")
	ErrInvalidPersonName    = errors.New("invalid person name")
	ErrInvalidPersonDesc    = errors.New("invalid person desc")
	ErrInvalidPersonURL     = errors.New("invalid person url")
)

func genwgKey(ruouterPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gen wg key: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, key[:], ruouterPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, key[:], shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("shuffler seal: %w", err)
	}

	pub := key.PublicKey()

	return pub[:], routerKey, shufflerKey, nil
}

func parseArgs() (bool, *tOpts, error) {
	var opts = &tOpts{}

	brigadeID := flag.String("id", "", "brigadier_id")
	brigadierName := flag.String("name", "", "brigadierName :: base64")
	endpointIPv4 := flag.String("ep4", "", "endpointIPv4")
	dnsIPv4 := flag.String("dns4", "", "dnsIPv4")
	dnsIPv6 := flag.String("dns6", "", "dnsIPv6")
	IPv4CGNAT := flag.String("int4", "", "IPv4CGNAT")
	IPv6ULA := flag.String("int6", "", "IPv6ULA")
	keydeskIPv6 := flag.String("kd6", "", "keydeskIPv6")
	personName := flag.String("person", "", "personName :: base64")
	personDesc := flag.String("desc", "", "personDesc :: base64")
	personURL := flag.String("url", "", "personURL :: base64")
	chunked := flag.Bool("ch", false, "chunked output")

	flag.Parse()

	// brigadeID must be base32 decodable.
	id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(*brigadeID)
	if err != nil {
		return false, nil, fmt.Errorf("id base32: %s: %w", *brigadeID, err)
	}

	_, err = uuid.FromBytes(id)
	if err != nil {
		return false, nil, fmt.Errorf("id uuid: %s: %w", *brigadeID, err)
	}

	opts.brigadeID = *brigadeID

	// brigadierName must be not empty and must be a valid UTF8 string
	if *brigadierName == "" {
		return false, nil, ErrEmptyBrigadierName
	}

	buf, err := base64.StdEncoding.DecodeString(*brigadierName)
	if err != nil {
		return false, nil, fmt.Errorf("brigadier name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, nil, ErrInvalidBrigadierName
	}

	opts.brigadierName = string(buf)

	// personName must be not empty and must be a valid UTF8 string
	if *personName == "" {
		return false, nil, ErrEmptyPersonName
	}

	buf, err = base64.StdEncoding.DecodeString(*personName)
	if err != nil {
		return false, nil, fmt.Errorf("person name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, nil, ErrInvalidPersonName
	}

	opts.person.Name = string(buf)

	// personDesc must be not empty and must be a valid UTF8 string
	if *personDesc == "" {
		return false, nil, ErrEmptyPersonDesc
	}

	buf, err = base64.StdEncoding.DecodeString(*personDesc)
	if err != nil {
		return false, nil, fmt.Errorf("person desc: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, nil, ErrInvalidPersonDesc
	}

	opts.person.Desc = string(buf)

	// personURL must be not empty and must be a valid UTF8 string
	if *personURL == "" {
		return false, nil, ErrEmptyPersonURL
	}

	buf, err = base64.StdEncoding.DecodeString(*personURL)
	if err != nil {
		return false, nil, fmt.Errorf("person url: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, nil, ErrInvalidPersonURL
	}

	u := string(buf)

	_, err = url.Parse(u)
	if err != nil {
		return false, nil, fmt.Errorf("parse person url: %w", err)
	}

	opts.person.URL = u

	// endpointIPv4 must be v4 and Global Unicast IP.
	ip, err := netip.ParseAddr(*endpointIPv4)
	if err != nil {
		return false, nil, fmt.Errorf("ep4: %s: %w", *endpointIPv4, err)
	}

	if !ip.Is4() || !ip.IsGlobalUnicast() {
		return false, nil, fmt.Errorf("ep4 ip4: %s: %w", ip, ErrInvalidEndpointIPv4)
	}

	opts.endpointIPv4 = ip

	// dnsIPv4 must be v4 IP
	ip, err = netip.ParseAddr(*dnsIPv4)
	if err != nil {
		return false, nil, fmt.Errorf("dns4: %s: %w", *dnsIPv4, err)
	}

	if !ip.Is4() {
		return false, nil, fmt.Errorf("dns4 ip4: %s: %w", ip, ErrInvalidDNS4)
	}

	opts.dnsIPv4 = ip

	// dnsIPv6 must be v6 IP
	ip, err = netip.ParseAddr(*dnsIPv6)
	if err != nil {
		return false, nil, fmt.Errorf("dns6: %s: %w", *dnsIPv6, err)
	}

	if !ip.Is6() {
		return false, nil, fmt.Errorf("dns6 ip6: %s: %w", ip, ErrInvalidDNS6)
	}

	opts.dnsIPv6 = ip

	cgnatPrefix := netip.MustParsePrefix(CGNATPrefix)

	// IPv4CGNAT must be v4 and private Network.
	pref, err := netip.ParsePrefix(*IPv4CGNAT)
	if err != nil {
		return false, nil, fmt.Errorf("int4: %s: %w", *IPv4CGNAT, err)
	}

	if cgnatPrefix.Bits() < pref.Bits() && !cgnatPrefix.Overlaps(pref) {
		return false, nil, fmt.Errorf("int4 ip4: %s: %w", ip, ErrInvalidNetCGNAT)
	}

	opts.IPv4CGNAT = pref

	ulaPrefix := netip.MustParsePrefix(ULAPrefix)

	// IPv6ULA must be v6 and private Network.
	pref, err = netip.ParsePrefix(*IPv6ULA)
	if err != nil {
		return false, nil, fmt.Errorf("int6: %s: %w", *IPv6ULA, err)
	}

	if ulaPrefix.Bits() < pref.Bits() && !ulaPrefix.Overlaps(pref) {
		return false, nil, fmt.Errorf("int6 ip6: %s: %w", ip, ErrInvalidNetULA)
	}

	opts.IPv6ULA = pref

	// keydeskIPv6 must be v6 and private Network.
	ip, err = netip.ParseAddr(*keydeskIPv6)
	if err != nil {
		return false, nil, fmt.Errorf("kd6: %s: %w", *keydeskIPv6, err)
	}

	if !ulaPrefix.Contains(ip) {
		return false, nil, fmt.Errorf("kd6 ip6: %s: %w", ip, ErrInvalidKeydeskIPv6)
	}

	opts.keydeskIPv6 = ip

	return *chunked, opts, nil
}

func do(opts *tOpts, wgPub, wgRouterPriv, wgShufflerPriv []byte) (string, string, error) {
	ctx := context.Background()

	tx, err := env.Env.DB.Begin(ctx)
	if err != nil {
		return "", "", fmt.Errorf("begin: %w", err)
	}

	brigadeNotify := &user.SrvNotify{
		T:        user.NotifyNewBrigade,
		Endpoint: user.NewEndpoint(opts.endpointIPv4),
		Brigade: user.SrvBrigade{
			ID:          env.Env.BrigadierID,
			IdentityKey: wgRouterPriv,
		},
	}

	notify, err := json.Marshal(brigadeNotify)
	if err != nil {
		tx.Rollback(context.Background())

		return "", "", fmt.Errorf("marshal notify: %w", err)
	}

	schema := (pgx.Identifier{env.Env.BrigadierID}).Sanitize()

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateSchema, schema))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create schema: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateRole, schema))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create role: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlTrimRole, schema))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("trim role: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlGrantRoleSchema, schema, schema))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("grant on schema to role: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlGrantRoleSchema, schema, vpngineUser))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("grant on schema to qrc: %w", err)
	}

	brigadeTable := (pgx.Identifier{env.Env.BrigadierID, "brigade"}).Sanitize()

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateTableBrigade, brigadeTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create brigade table: %w", err)
	}

	usersTable := (pgx.Identifier{env.Env.BrigadierID, "users"}).Sanitize()

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateTableUsers, usersTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create users table: %w", err)
	}

	quotaTable := (pgx.Identifier{env.Env.BrigadierID, "quota"}).Sanitize()

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateTableQuota, quotaTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create quota table: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlAlterTableQuota, quotaTable, usersTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("alter quota table: %w", err)
	}

	personsTable := (pgx.Identifier{env.Env.BrigadierID, "persons"}).Sanitize()

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateTablePersons, personsTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create persons table: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlAlterTablePersons, personsTable, usersTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("alter persons table: %w", err)
	}

	keydeskTable := (pgx.Identifier{env.Env.BrigadierID, "keydesk"}).Sanitize()

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlCreateTableKeydesk, keydeskTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("create keydesk table: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlInsertBrigade, brigadeTable),
		`\x`+hex.EncodeToString(wgPub),
		`\x`+hex.EncodeToString(wgRouterPriv),
		`\x`+hex.EncodeToString(wgShufflerPriv),
		opts.endpointIPv4.String(),
		opts.dnsIPv4.String(),
		opts.dnsIPv6.String(),
		opts.keydeskIPv6.String(),
		opts.IPv4CGNAT.String(),
		opts.IPv6ULA.String(),
	)
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("insert brigade: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlInsertKeydesk, keydeskTable))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("insert keydesk: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlGrantRoleTables, schema, schema))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("grant on tables to role: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(sqlGrantRoleTables, schema, vpngineUser))
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("grant on tables to qrc: %w", err)
	}

	_, err = tx.Exec(ctx, sqlNotifyNewBrigade, notify)
	if err != nil {
		tx.Rollback(ctx)

		return "", "", fmt.Errorf("notify: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return "", "", fmt.Errorf("commit: %w", err)
	}

	wgconf, filename, err := user.AddBrigadier(opts.brigadierName, opts.person)
	if err != nil {
		return "", "", fmt.Errorf("create brigadier: %w", err)
	}

	return wgconf, filename, nil
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

	chunked, opts, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	env.Env.BrigadierID = opts.brigadeID

	err = env.CreatePool()
	if err != nil {
		log.Fatalln(err)
	}

	defer env.Env.DB.Close()

	wgPub, wgRouterPriv, wgShufflerPriv, err := genwgKey(&env.Env.RouterPublicKey, &env.Env.ShufflerPublicKey)
	if err != nil {
		log.Fatalf("Can't create wg keys: %s", err)
	}

	// just do it.
	wgconf, filename, err := do(opts, wgPub, wgRouterPriv, wgShufflerPriv)
	if err != nil {
		log.Fatalf("Can't create brigade: %s", err)
	}

	switch chunked {
	case true:
		w = httputil.NewChunkedWriter(os.Stdout)
		defer w.Close()
	default:
		w = os.Stdout
	}

	_, err = fmt.Fprintln(w, filename)
	if err != nil {
		log.Fatalf("Can't print filename: %s\n", err)
	}

	_, err = fmt.Fprintln(w, wgconf)
	if err != nil {
		log.Fatalf("Can't print wgconf: %s\n", err)
	}
}
