package main

import (
	"encoding/base32"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/coreos/go-systemd/activation"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
	"github.com/vpngen/wordsgens/namesgenerator"
	"log"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type flags struct {
	webDir    *string
	etcDir    *string
	filedbDir *string
	certDir   *string
	statsDir  *string

	pcors      *bool
	brigadeID  *string
	listenAddr *string

	brigadierName    *string
	personName       *string
	personDesc       *string
	personURL        *string
	replaceBrigadier *bool

	addr *string

	chunked *bool
	jsonOut *bool

	wgcCfgs     *string
	ovcCfgs     *string
	ipsecCfgs   *string
	outlineCfgs *string

	messageAPI *string
}

const defaultMsgSocketDir = "/var/lib/dcapi"

func parseFlags(flagSet *flag.FlagSet, args []string) flags {
	var f flags

	f.webDir = flagSet.String("w", DefaultWebDir, "Dir for web files.")
	f.etcDir = flagSet.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	f.filedbDir = flagSet.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	f.certDir = flagSet.String("e", "", "Dir for TLS certificate and key (for test). Default: "+DefaultCertDir)
	f.statsDir = flagSet.String("s", "", "Dir with brigades statistics. Default: "+storage.DefaultStatsDir+"/<BrigadeID>")

	f.pcors = flagSet.Bool("cors", false, "Turn on permessive CORS (for test)")
	f.brigadeID = flagSet.String("id", "", "BrigadeID (for test)")
	f.listenAddr = flagSet.String("l", "", "Listen addr:port (http and https separate with commas)")

	f.brigadierName = flagSet.String("name", "", "brigadierName :: base64")
	f.personName = flagSet.String("person", "", "personName :: base64")
	f.personDesc = flagSet.String("desc", "", "personDesc :: base64")
	f.personURL = flagSet.String("url", "", "personURL :: base64")
	f.replaceBrigadier = flagSet.Bool("r", false, "Replace brigadier config")

	f.addr = flagSet.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")

	f.chunked = flagSet.Bool("ch", false, "chunked output")
	f.jsonOut = flagSet.Bool("j", false, "json output")

	f.wgcCfgs = flagSet.String("wg", "native,amnezia", "Wireguard configs ("+storage.ConfigsWg+")")
	f.ovcCfgs = flagSet.String("ovc", "", "OpenVPN over Cloak configs ("+storage.ConfigsOvc+")")
	f.ipsecCfgs = flagSet.String("ipsec", "", "IPSec configs ("+storage.ConfigsIPSec+")")
	f.outlineCfgs = flagSet.String("outline", "", "Outline configs ("+storage.ConfigsOutline+")")

	f.messageAPI = flagSet.String("m", "", fmt.Sprintf("Message API unix socket path. Default: %s/<BrigadeID>.sock '-' to disable", defaultMsgSocketDir))

	// ignore errors, see original flag.Parse() func
	_ = flagSet.Parse(args)

	return f
}

type config struct {
	chunked          bool
	jsonOut          bool
	enableCORS       bool
	listeners        []net.Listener
	addr             netip.AddrPort
	brigadeID        string
	etcDir           string
	webDir           string
	dbDir            string
	certDir          string
	statsDir         string
	brigadierName    string
	person           namesgenerator.Person
	replaceBrigadier bool
	vpnConfigs       *storage.ConfigsImplemented
	messageAPISocket net.Listener
}

func parseArgs2(flags flags) (config, error) {
	cfg := config{
		chunked:    *flags.chunked,
		jsonOut:    *flags.jsonOut,
		enableCORS: *flags.pcors,
	}

	log.Println("getting user")
	sysUser, err := user.Current()
	if err != nil {
		return cfg, fmt.Errorf("cannot define user: %w", err)
	}

	log.Println("parsing VPN configs")
	cfg.vpnConfigs = parseVPNConfigs(flags)

	if *flags.webDir == "" {
		return cfg, ErrStaticDirEmpty
	}

	log.Println("getting dirs")
	cfg, err = getAbsDirPaths(cfg, flags)
	if err != nil {
		return cfg, err
	}

	log.Println("getting brigadeID")
	switch *flags.brigadeID {
	case "", sysUser.Username:
		cfg.brigadeID = sysUser.Username
		cfg = setDefaultDirs(flags, cfg)
	default:
		cfg.brigadeID = *flags.brigadeID
		cfg, err = setDirsCWD(flags, cfg)
		if err != nil {
			return cfg, err
		}
	}

	if err = checkBase32EncodedUUID(cfg.brigadeID); err != nil {
		return cfg, err
	}

	log.Println("getting addr")
	if *flags.addr != "-" {
		cfg.addr, err = netip.ParseAddrPort(*flags.addr)
		if err != nil {
			return cfg, fmt.Errorf("api addr: %w", err)
		}
	}

	if *flags.replaceBrigadier {
		cfg.replaceBrigadier = true
		return cfg, nil
	}

	log.Println("getting message API socket")
	cfg, err = parseMessageAPISocket(flags, cfg)
	if err != nil {
		return cfg, err
	}

	log.Println("getting listeners")
	if *flags.brigadierName == "" {
		switch *flags.listenAddr {
		case "":
			// get listeners from activation sockets
			cfg.listeners, err = activation.Listeners()
			if err != nil {
				return cfg, fmt.Errorf("cannot retrieve listeners: %w", err)
			}

			return cfg, nil
		default:
			// get listeners from argument
			for _, laddr := range strings.Split(*flags.listenAddr, ",") {
				l, err := net.Listen("tcp", laddr)
				if err != nil {
					return cfg, fmt.Errorf("cannot listen: %w", err)
				}

				cfg.listeners = append(cfg.listeners, l)
			}

			if len(cfg.listeners) != 1 && len(cfg.listeners) != 2 {
				return cfg, fmt.Errorf("unexpected number of litening (%d != 1|2)", len(cfg.listeners))
			}
		}

		return cfg, nil
	}

	if *flags.brigadierName != "" {
		cfg.brigadierName, err = decodeBas64AndCheck(*flags.brigadierName)
		if err != nil {
			return cfg, ErrInvalidBrigadierName
		}
	}

	cfg, err = parsePerson(flags, cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func parseMessageAPISocket(f flags, cfg config) (config, error) {
	var path string

	switch *f.messageAPI {
	case "-":
		return cfg, nil
	case "":
		path = defaultMsgSocketDir + "/" + cfg.brigadeID + ".sock"
	default:
		path = *f.messageAPI
	}

	log.Println("Message API socket path:", path)
	// delete socket if exists
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		if err := os.Remove(path); err != nil {
			return cfg, fmt.Errorf("cannot remove socket: %w", err)
		}
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return cfg, fmt.Errorf("cannot listen: %w", err)
	}

	cfg.messageAPISocket = l

	return cfg, nil
}

func parseVPNConfigs(flags flags) *storage.ConfigsImplemented {
	vpnCfgs := storage.NewConfigsImplemented()

	if *flags.wgcCfgs != "" {
		vpnCfgs.AddWg(*flags.wgcCfgs)
	}

	if *flags.ovcCfgs != "" {
		vpnCfgs.AddOvc(*flags.ovcCfgs)
	}

	if *flags.ipsecCfgs != "" {
		vpnCfgs.AddIPSec(*flags.ipsecCfgs)
	}

	if *flags.outlineCfgs != "" {
		vpnCfgs.AddOutline(*flags.outlineCfgs)
	}

	return vpnCfgs
}

func getAbsDirPaths(cfg config, flags flags) (config, error) {
	var err error

	cfg.webDir, err = filepath.Abs(*flags.webDir)
	if err != nil {
		return cfg, fmt.Errorf("web dir: %w", err)
	}

	if *flags.filedbDir != "" {
		cfg.dbDir, err = filepath.Abs(*flags.filedbDir)
		if err != nil {
			return cfg, fmt.Errorf("db dir: %w", err)
		}
	}

	if *flags.etcDir != "" {
		cfg.etcDir, err = filepath.Abs(*flags.etcDir)
		if err != nil {
			return cfg, fmt.Errorf("etc dir: %w", err)
		}
	}

	if *flags.certDir != "" {
		cfg.certDir, err = filepath.Abs(*flags.certDir)
		if err != nil {
			return cfg, fmt.Errorf("cert dir: %w", err)
		}
	}

	if *flags.statsDir != "" {
		cfg.statsDir, err = filepath.Abs(*flags.statsDir)
		if err != nil {
			return cfg, fmt.Errorf("stat dir: %w", err)
		}
	}

	return cfg, nil
}

func setDefaultDirs(flags flags, cfg config) config {
	if *flags.filedbDir == "" {
		cfg.dbDir = filepath.Join(storage.DefaultHomeDir, cfg.brigadeID)
	}

	if *flags.etcDir == "" {
		cfg.etcDir = keydesk.DefaultEtcDir
	}

	if *flags.certDir == "" {
		cfg.certDir = DefaultCertDir
	}

	if *flags.statsDir == "" {
		cfg.statsDir = filepath.Join(storage.DefaultStatsDir, cfg.brigadeID)
	}

	return cfg
}

func setDirsCWD(flags flags, cfg config) (config, error) {
	cwd, err := os.Getwd()
	/*
		TODO
		do we have to handle this error?
		if err != nil {
			return cfg, fmt.Errorf("get cwd: %w", err)
		}
		would this break anything?
	*/
	if err == nil {
		cwd, err = filepath.Abs(cwd)
		if err != nil {
			return cfg, fmt.Errorf("get abs cwd: %w", err)
		}
	}

	if *flags.filedbDir == "" {
		cfg.dbDir = cwd
	}

	if *flags.etcDir == "" {
		cfg.etcDir = cwd
	}

	if *flags.certDir == "" {
		cfg.certDir = cwd
	}

	if *flags.statsDir == "" {
		cfg.statsDir = cwd
	}

	return cfg, nil
}

func checkBase32EncodedUUID(s string) error {
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return fmt.Errorf("id base32: %s: %w", s, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return fmt.Errorf("id uuid: %s: %w", s, err)
	}

	return nil
}

func decodeBas64AndCheck(s string) (string, error) {
	buf, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s, fmt.Errorf("person name: %w", err)
	}

	if !utf8.Valid(buf) {
		return s, fmt.Errorf("invalid utf8")
	}

	return string(buf), nil
}

func parsePerson(flags flags, cfg config) (config, error) {
	var err error

	if *flags.personName != "" {
		cfg.person.Name, err = decodeBas64AndCheck(*flags.personName)
		if err != nil {
			return cfg, ErrInvalidPersonName
		}
	}

	if *flags.personDesc != "" {
		cfg.person.Desc, err = decodeBas64AndCheck(*flags.personDesc)
		if err != nil {
			return cfg, ErrInvalidPersonDesc
		}
	}

	if *flags.personURL != "" {
		cfg.person.URL, err = decodeBas64AndCheck(*flags.personURL)
		if err != nil {
			return cfg, ErrInvalidPersonURL
		}

		_, err = url.Parse(cfg.person.URL)
		if err != nil {
			return cfg, fmt.Errorf("parse person url: %w", err)
		}
	}

	return cfg, nil
}

func parseArgs(flags flags) (bool, bool, bool, []net.Listener, netip.AddrPort, string, string, string, string, string, string, string, namesgenerator.Person, bool, *storage.ConfigsImplemented, error) {
	var (
		id                               string
		etcdir, dbdir, certdir, statsdir string
		person                           namesgenerator.Person
		addrPort                         netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot define user: %w", err)
	}

	vpnCfgs := storage.NewConfigsImplemented()

	if *flags.wgcCfgs != "" {
		vpnCfgs.AddWg(*flags.wgcCfgs)
	}

	if *flags.ovcCfgs != "" {
		vpnCfgs.AddOvc(*flags.ovcCfgs)
	}

	if *flags.ipsecCfgs != "" {
		vpnCfgs.AddIPSec(*flags.ipsecCfgs)
	}

	if *flags.outlineCfgs != "" {
		vpnCfgs.AddOutline(*flags.outlineCfgs)
	}

	if *flags.webDir == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrStaticDirEmpty
	}

	webdir, err := filepath.Abs(*flags.webDir)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("web dir: %w", err)
	}

	if *flags.filedbDir != "" {
		dbdir, err = filepath.Abs(*flags.filedbDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *flags.etcDir != "" {
		etcdir, err = filepath.Abs(*flags.etcDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("etcdir dir: %w", err)
		}
	}

	if *flags.certDir != "" {
		certdir, err = filepath.Abs(*flags.certDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("certdir dir: %w", err)
		}
	}

	if *flags.statsDir != "" {
		statsdir, err = filepath.Abs(*flags.statsDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("statdir dir: %w", err)
		}
	}

	switch *flags.brigadeID {
	case "", sysUser.Username:
		id = sysUser.Username

		if *flags.filedbDir == "" {
			dbdir = filepath.Join(storage.DefaultHomeDir, id)
		}

		if *flags.etcDir == "" {
			etcdir = keydesk.DefaultEtcDir
		}

		if *flags.certDir == "" {
			certdir = DefaultCertDir
		}

		if *flags.statsDir == "" {
			statsdir = filepath.Join(storage.DefaultStatsDir, id)
		}
	default:
		id = *flags.brigadeID

		cwd, err := os.Getwd()
		if err == nil {
			cwd, _ = filepath.Abs(cwd)
		}

		if *flags.filedbDir == "" {
			dbdir = cwd
		}

		if *flags.etcDir == "" {
			etcdir = cwd
		}

		if *flags.certDir == "" {
			certdir = cwd
		}

		if *flags.statsDir == "" {
			statsdir = cwd
		}
	}

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("id uuid: %s: %w", id, err)
	}

	if *flags.addr != "-" {
		addrPort, err = netip.ParseAddrPort(*flags.addr)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("api addr: %w", err)
		}
	}

	if *flags.replaceBrigadier {
		return *flags.chunked, *flags.jsonOut, *flags.pcors, nil, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, "", person, *flags.replaceBrigadier, vpnCfgs, nil
	}

	if *flags.brigadierName == "" {
		var listeners []net.Listener

		switch *flags.listenAddr {
		case "":
			// get listeners from activation sockets
			listeners, err = activation.Listeners()
			if err != nil {
				return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot retrieve listeners: %w", err)
			}

			return *flags.chunked, *flags.jsonOut, *flags.pcors, listeners, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, "", person, false, nil, nil
		default:
			// get listeners from argument
			for _, laddr := range strings.Split(*flags.listenAddr, ",") {
				l, err := net.Listen("tcp", laddr)
				if err != nil {
					return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot listen: %w", err)
				}

				listeners = append(listeners, l)
			}

			if len(listeners) != 1 && len(listeners) != 2 {
				return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("unexpected number of litening (%d != 1|2)",
					len(listeners))
			}
		}

		return *flags.chunked, *flags.jsonOut, *flags.pcors, listeners, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, "", person, false, nil, nil
	}

	// brigadierName must be not empty and must be a valid UTF8 string
	buf, err := base64.StdEncoding.DecodeString(*flags.brigadierName)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("brigadier name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidBrigadierName
	}

	name := string(buf)

	// personName must be not empty and must be a valid UTF8 string
	if *flags.personName == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrEmptyPersonName
	}

	buf, err = base64.StdEncoding.DecodeString(*flags.personName)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("person name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidPersonName
	}

	person.Name = string(buf)

	// personDesc must be not empty and must be a valid UTF8 string
	if *flags.personDesc == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrEmptyPersonDesc
	}

	buf, err = base64.StdEncoding.DecodeString(*flags.personDesc)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("person desc: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidPersonDesc
	}

	person.Desc = string(buf)

	// personURL must be not empty and must be a valid UTF8 string
	if *flags.personURL == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrEmptyPersonURL
	}

	buf, err = base64.StdEncoding.DecodeString(*flags.personURL)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("person url: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidPersonURL
	}

	u := string(buf)

	_, err = url.Parse(u)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("parse person url: %w", err)
	}

	person.URL = u

	return *flags.chunked, *flags.jsonOut, *flags.pcors, nil, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, name, person, *flags.replaceBrigadier, vpnCfgs, nil
}
