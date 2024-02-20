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
}

func parseFlags() flags {
	var f flags

	f.webDir = flag.String("w", DefaultWebDir, "Dir for web files.")
	f.etcDir = flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	f.filedbDir = flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	f.certDir = flag.String("e", "", "Dir for TLS certificate and key (for test). Default: "+DefaultCertDir)
	f.statsDir = flag.String("s", "", "Dir with brigades statistics. Default: "+storage.DefaultStatsDir+"/<BrigadeID>")

	f.pcors = flag.Bool("cors", false, "Turn on permessive CORS (for test)")
	f.brigadeID = flag.String("id", "", "BrigadeID (for test)")
	f.listenAddr = flag.String("l", "", "Listen addr:port (http and https separate with commas)")

	f.brigadierName = flag.String("name", "", "brigadierName :: base64")
	f.personName = flag.String("person", "", "personName :: base64")
	f.personDesc = flag.String("desc", "", "personDesc :: base64")
	f.personURL = flag.String("url", "", "personURL :: base64")
	f.replaceBrigadier = flag.Bool("r", false, "Replace brigadier config")

	f.addr = flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")

	f.chunked = flag.Bool("ch", false, "chunked output")
	f.jsonOut = flag.Bool("j", false, "json output")

	f.wgcCfgs = flag.String("wg", "native,amnezia", "Wireguard configs ("+storage.ConfigsWg+")")
	f.ovcCfgs = flag.String("ovc", "", "OpenVPN over Cloak configs ("+storage.ConfigsOvc+")")
	f.ipsecCfgs = flag.String("ipsec", "", "IPSec configs ("+storage.ConfigsIPSec+")")
	f.outlineCfgs = flag.String("outline", "", "Outline configs ("+storage.ConfigsOutline+")")

	flag.Parse()

	return f
}

type config struct {
	chunked          bool
	jsonOut          bool
	enableCORS       bool
	listeners        []net.Listener
	addrPort         netip.AddrPort
	id               string
	etcDir           string
	webDir           string
	dbDir            string
	certDir          string
	statsDir         string
	brigadierName    string
	person           namesgenerator.Person
	replaceBrigadier bool
	vpnConfigs       *storage.ConfigsImplemented
}

func parseArgs2(flags flags) (config, error) {
	var cfg config

	cfg.chunked = *flags.chunked
	cfg.jsonOut = *flags.jsonOut
	cfg.enableCORS = *flags.pcors

	sysUser, err := user.Current()
	if err != nil {
		return cfg, fmt.Errorf("cannot define user: %w", err)
	}

	cfg.vpnConfigs = parseVPNConfigs(flags)

	if *flags.webDir == "" {
		return cfg, ErrStaticDirEmpty
	}

	cfg, err = getAbsDirPaths(cfg, flags)
	if err != nil {
		return cfg, err
	}

	switch *flags.brigadeID {
	case "", sysUser.Username:
		cfg.id = sysUser.Username
		cfg = setDefaultDirs(flags, cfg)
	default:
		cfg.id = *flags.brigadeID
		cfg, err = setDirsCWD(flags, cfg)
		if err != nil {
			return cfg, err
		}
	}

	if err = checkBase32EncodedUUID(cfg.id); err != nil {
		return cfg, err
	}

	if *flags.addr != "-" {
		cfg.addrPort, err = netip.ParseAddrPort(*flags.addr)
		if err != nil {
			return cfg, fmt.Errorf("api addr: %w", err)
		}
	}

	/*
		TODO
		do we have to return here?
	*/
	if *flags.replaceBrigadier {
		cfg.replaceBrigadier = true
		return cfg, nil
	}

	if *flags.brigadierName == "" {
		var listeners []net.Listener

		switch *flags.listenAddr {
		case "":
			// get listeners from activation sockets
			listeners, err = activation.Listeners()
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

				listeners = append(listeners, l)
			}

			if len(listeners) != 1 && len(listeners) != 2 {
				return cfg, fmt.Errorf("unexpected number of litening (%d != 1|2)", len(listeners))
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
		cfg.dbDir = filepath.Join(storage.DefaultHomeDir, cfg.id)
	}

	if *flags.etcDir == "" {
		cfg.etcDir = keydesk.DefaultEtcDir
	}

	if *flags.certDir == "" {
		cfg.certDir = DefaultCertDir
	}

	if *flags.statsDir == "" {
		cfg.statsDir = filepath.Join(storage.DefaultStatsDir, cfg.id)
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
