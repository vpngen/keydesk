package main

import (
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
	"github.com/vpngen/vpngine/naclkey"
)

// Args errors.
var (
	ErrInvalidEndpointIPv4 = errors.New("invalid ip4 endpoint")
	ErrInvalidDNS4         = errors.New("invalid dns ip4 endpoint")
	ErrInvalidDNS6         = errors.New("invalid dns ip6 endpoint")
	ErrInvalidNetCGNAT     = errors.New("invalid cgnat net")
	ErrInvalidNetULA       = errors.New("invalid ula net")
	ErrInvalidKeydeskIPv6  = errors.New("invalid keydesk ip6 endpoint")
)

func parseArgs() (*storage.BrigadeConfig, netip.AddrPort, string, string, error) {
	var (
		config        = &storage.BrigadeConfig{}
		dbdir, etcdir string
		id            string
		addrPort      netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("cannot define user: %w", err)
	}

	endpointIPv4 := flag.String("ep4", "", "endpointIPv4")
	dnsIPv4 := flag.String("dns4", "", "dnsIPv4")
	dnsIPv6 := flag.String("dns6", "", "dnsIPv6")
	IPv4CGNAT := flag.String("int4", "", "IPv4CGNAT")
	IPv6ULA := flag.String("int6", "", "IPv6ULA")
	keydeskIPv6 := flag.String("kd6", "", "keydeskIPv6")
	// !!! is the flags below only for debug?
	brigadeID := flag.String("id", "", "brigadier_id")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")

	flag.Parse()

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return nil, addrPort, "", "", fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return nil, addrPort, "", "", fmt.Errorf("etcdir dir: %w", err)
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

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("id uuid: %s: %w", id, err)
	}

	config.BrigadeID = id

	// endpointIPv4 must be v4 and Global Unicast IP.
	ip, err := netip.ParseAddr(*endpointIPv4)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("ep4: %s: %w", *endpointIPv4, err)
	}

	if !ip.Is4() || !ip.IsGlobalUnicast() {
		return nil, addrPort, "", "", fmt.Errorf("ep4 ip4: %s: %w", ip, ErrInvalidEndpointIPv4)
	}

	config.EndpointIPv4 = ip

	// dnsIPv4 must be v4 IP
	ip, err = netip.ParseAddr(*dnsIPv4)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("dns4: %s: %w", *dnsIPv4, err)
	}

	if !ip.Is4() {
		return nil, addrPort, "", "", fmt.Errorf("dns4 ip4: %s: %w", ip, ErrInvalidDNS4)
	}

	config.DNSIPv4 = ip

	// dnsIPv6 must be v6 IP
	ip, err = netip.ParseAddr(*dnsIPv6)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("dns6: %s: %w", *dnsIPv6, err)
	}

	if !ip.Is6() {
		return nil, addrPort, "", "", fmt.Errorf("dns6 ip6: %s: %w", ip, ErrInvalidDNS6)
	}

	config.DNSIPv6 = ip

	cgnatPrefix := netip.MustParsePrefix(keydesk.CGNATPrefix)

	// IPv4CGNAT must be v4 and private Network.
	pref, err := netip.ParsePrefix(*IPv4CGNAT)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("int4: %s: %w", *IPv4CGNAT, err)
	}

	if cgnatPrefix.Bits() < pref.Bits() && !cgnatPrefix.Overlaps(pref) {
		return nil, addrPort, "", "", fmt.Errorf("int4 ip4: %s: %w", ip, ErrInvalidNetCGNAT)
	}

	config.IPv4CGNAT = pref

	ulaPrefix := netip.MustParsePrefix(keydesk.ULAPrefix)

	// IPv6ULA must be v6 and private Network.
	pref, err = netip.ParsePrefix(*IPv6ULA)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("int6: %s: %w", *IPv6ULA, err)
	}

	if ulaPrefix.Bits() < pref.Bits() && !ulaPrefix.Overlaps(pref) {
		return nil, addrPort, "", "", fmt.Errorf("int6 ip6: %s: %w", ip, ErrInvalidNetULA)
	}

	config.IPv6ULA = pref

	// keydeskIPv6 must be v6 and private Network.
	ip, err = netip.ParseAddr(*keydeskIPv6)
	if err != nil {
		return nil, addrPort, "", "", fmt.Errorf("kd6: %s: %w", *keydeskIPv6, err)
	}

	if !ulaPrefix.Contains(ip) {
		return nil, addrPort, "", "", fmt.Errorf("kd6 ip6: %s: %w", ip, ErrInvalidKeydeskIPv6)
	}

	config.KeydeskIPv6 = ip

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return nil, addrPort, "", "", fmt.Errorf("api addr: %w", err)
		}
	}

	return config, addrPort, etcdir, dbdir, nil
}

func main() {
	config, addr, etcDir, dbDir, err := parseArgs()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Can't parse args: %s", err)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       config.BrigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:              keydesk.MaxUsers,
			MonthlyQuotaRemaining: keydesk.MonthlyQuotaRemaining,
			ActivityPeriod:        keydesk.ActivityPeriod,
		},
	}
	if err := db.SelfCheck(); err != nil {
		log.Fatalf("Storage check error: %s", err)
	}

	// just do it.
	if err := keydesk.CreateBrigade(db, config, &routerPublicKey, &shufflerPublicKey); err != nil {
		log.Fatalf("Can't create brigade: %s\n", err)
	}
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
