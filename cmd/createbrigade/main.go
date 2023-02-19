package main

import (
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os/user"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/vpngen/keydesk/epapi"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
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

func parseArgs() (*storage.BrigadeConfig, netip.AddrPort, string, string, string, error) {
	var config = &storage.BrigadeConfig{}

	addrPort := netip.AddrPort{}

	endpointIPv4 := flag.String("ep4", "", "endpointIPv4")
	dnsIPv4 := flag.String("dns4", "", "dnsIPv4")
	dnsIPv6 := flag.String("dns6", "", "dnsIPv6")
	IPv4CGNAT := flag.String("int4", "", "IPv4CGNAT")
	IPv6ULA := flag.String("int6", "", "IPv6ULA")
	keydeskIPv6 := flag.String("kd6", "", "keydeskIPv6")
	// !!! is the flags below only for debug?
	brigadeID := flag.String("id", "", "brigadier_id")
	etcDir := flag.String("c", keydesk.DefaultEtcDir, "Dir for config files (for test)")
	homeDir := flag.String("d", storage.DefaultFileDbDir, "Dir for db files (for test)")
	statDir := flag.String("s", storage.DefaultStatDir, "Dir for stst files (for test)")
	addr := flag.String("a", epapi.TemplatedAddrPort, "API endpoint address:port")

	flag.Parse()

	if *brigadeID == "" {
		username, err := user.Current()
		if err != nil {
			return nil, addrPort, "", "", "", fmt.Errorf("username: %w", err)
		}

		brigadeID = &username.Username
	}

	// brigadeID must be base32 decodable.
	id, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(*brigadeID)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("id base32: %s: %w", *brigadeID, err)
	}

	_, err = uuid.FromBytes(id)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("id uuid: %s: %w", *brigadeID, err)
	}

	config.BrigadeID = *brigadeID

	if *homeDir == "" {
		*homeDir = filepath.Join("home", config.BrigadeID)
	}

	dbdir, err := filepath.Abs(*homeDir)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("dbdir dir: %w", err)
	}

	etcdir, err := filepath.Abs(*etcDir)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("etcdir dir: %w", err)
	}

	statdir, err := filepath.Abs(*statDir)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("statdir dir: %w", err)
	}

	// endpointIPv4 must be v4 and Global Unicast IP.
	ip, err := netip.ParseAddr(*endpointIPv4)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("ep4: %s: %w", *endpointIPv4, err)
	}

	if !ip.Is4() || !ip.IsGlobalUnicast() {
		return nil, addrPort, "", "", "", fmt.Errorf("ep4 ip4: %s: %w", ip, ErrInvalidEndpointIPv4)
	}

	config.EndpointIPv4 = ip

	// dnsIPv4 must be v4 IP
	ip, err = netip.ParseAddr(*dnsIPv4)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("dns4: %s: %w", *dnsIPv4, err)
	}

	if !ip.Is4() {
		return nil, addrPort, "", "", "", fmt.Errorf("dns4 ip4: %s: %w", ip, ErrInvalidDNS4)
	}

	config.DNSIPv4 = ip

	// dnsIPv6 must be v6 IP
	ip, err = netip.ParseAddr(*dnsIPv6)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("dns6: %s: %w", *dnsIPv6, err)
	}

	if !ip.Is6() {
		return nil, addrPort, "", "", "", fmt.Errorf("dns6 ip6: %s: %w", ip, ErrInvalidDNS6)
	}

	config.DNSIPv6 = ip

	cgnatPrefix := netip.MustParsePrefix(keydesk.CGNATPrefix)

	// IPv4CGNAT must be v4 and private Network.
	pref, err := netip.ParsePrefix(*IPv4CGNAT)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("int4: %s: %w", *IPv4CGNAT, err)
	}

	if cgnatPrefix.Bits() < pref.Bits() && !cgnatPrefix.Overlaps(pref) {
		return nil, addrPort, "", "", "", fmt.Errorf("int4 ip4: %s: %w", ip, ErrInvalidNetCGNAT)
	}

	config.IPv4CGNAT = pref

	ulaPrefix := netip.MustParsePrefix(keydesk.ULAPrefix)

	// IPv6ULA must be v6 and private Network.
	pref, err = netip.ParsePrefix(*IPv6ULA)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("int6: %s: %w", *IPv6ULA, err)
	}

	if ulaPrefix.Bits() < pref.Bits() && !ulaPrefix.Overlaps(pref) {
		return nil, addrPort, "", "", "", fmt.Errorf("int6 ip6: %s: %w", ip, ErrInvalidNetULA)
	}

	config.IPv6ULA = pref

	// keydeskIPv6 must be v6 and private Network.
	ip, err = netip.ParseAddr(*keydeskIPv6)
	if err != nil {
		return nil, addrPort, "", "", "", fmt.Errorf("kd6: %s: %w", *keydeskIPv6, err)
	}

	if !ulaPrefix.Contains(ip) {
		return nil, addrPort, "", "", "", fmt.Errorf("kd6 ip6: %s: %w", ip, ErrInvalidKeydeskIPv6)
	}

	config.KeydeskIPv6 = ip

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return nil, addrPort, "", "", "", fmt.Errorf("api addr: %w", err)
		}
	}

	return config, addrPort, etcdir, dbdir, statdir, nil
}

func main() {
	config, addr, etcDir, dbDir, statDir, err := parseArgs()
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
		StatsFilename:   filepath.Join(statDir, fmt.Sprintf(storage.StatFilename, config.BrigadeID)),
		APIAddrPort:     addr,
	}

	// just do it.
	err = keydesk.CreateBrigade(db, config, &routerPublicKey, &shufflerPublicKey)
	if err != nil {
		log.Fatalf("Can't create brigade: %s", err)
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
