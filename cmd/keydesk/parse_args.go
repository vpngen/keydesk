package main

import (
	"encoding/base32"
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vpnapi"
	"github.com/vpngen/wordsgens/namesgenerator"
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
	proto0Cfgs  *string

	unixSocketDir    *string
	messageAPI       *string
	shufflerAPI      *string
	jwtPublicKeyFile *string
}

const (
	defaultUnixSocketDir  = "/var/lib/dcapi"
	defaultMessageSocket  = "messages.sock"
	defaultShufflerSocket = "shuffler.sock"
	jwtPubFileName        = "jwt-pub-msg.pem"
)

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
	f.proto0Cfgs = flagSet.String("proto0", "", "Protocol0 configs ("+storage.ConfigsProto0+")")

	f.unixSocketDir = flagSet.String("socket-dir", defaultUnixSocketDir, fmt.Sprintf("Unix sockets dir. Default: %s", defaultUnixSocketDir))
	f.messageAPI = flagSet.String("m", "", fmt.Sprintf("Message API unix socket path. Default: %s/<BrigadeID>/messages.sock '-' to disable", *f.unixSocketDir))
	f.shufflerAPI = flagSet.String("shuffler", "", fmt.Sprintf("Shuffler API unix socket path. Default: %s/<BrigadeID>/shuffler.sock '-' to disable", *f.unixSocketDir))
	f.jwtPublicKeyFile = flagSet.String("jwtpub", "", fmt.Sprintf("Path to JWT public key file. Default: %s/%s", keydesk.DefaultEtcDir, jwtPubFileName))

	// ignore errors, see original flag.Parse() func
	_ = flagSet.Parse(args)

	return f
}

type config struct {
	chunked           bool
	jsonOut           bool
	enableCORS        bool
	listeners         []net.Listener
	addr              netip.AddrPort
	brigadeID         string
	etcDir            string
	webDir            string
	dbDir             string
	certDir           string
	statsDir          string
	brigadierName     string
	person            namesgenerator.Person
	replaceBrigadier  bool
	vpnConfigs        *storage.ConfigsImplemented
	unixSocketDir     string
	messageAPISocket  net.Listener
	shufflerAPISocket net.Listener
	jwtPublicKeyFile  string
}

func parseArgs2(flags flags) (config, error) {
	cfg := config{
		chunked:       *flags.chunked,
		jsonOut:       *flags.jsonOut,
		enableCORS:    *flags.pcors,
		unixSocketDir: *flags.unixSocketDir,
	}

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

	if *flags.brigadierName == "" {
		if *flags.jwtPublicKeyFile == "" {
			cfg.jwtPublicKeyFile = filepath.Join(cfg.etcDir, jwtPubFileName)
		}

		listener, err := createUnixSocketListener(*flags.messageAPI, cfg.brigadeID, cfg.unixSocketDir, defaultMessageSocket)
		if err != nil {
			return cfg, fmt.Errorf("create messages listener: %w", err)
		}
		cfg.messageAPISocket = listener

		listener, err = createUnixSocketListener(*flags.shufflerAPI, cfg.brigadeID, cfg.unixSocketDir, defaultShufflerSocket)
		if err != nil {
			return cfg, fmt.Errorf("create shuffler listener: %w", err)
		}
		cfg.shufflerAPISocket = listener

		// get listeners from argument
		for _, laddr := range strings.Split(*flags.listenAddr, ",") {
			if laddr == "" {
				continue
			}
			l, err := net.Listen("tcp", laddr)
			if err != nil {
				return cfg, fmt.Errorf("cannot listen: %w", err)
			}

			cfg.listeners = append(cfg.listeners, l)
		}

		//if len(cfg.listeners) != 1 && len(cfg.listeners) != 2 {
		//	return cfg, fmt.Errorf("unexpected number of litening (%d != 1|2)", len(cfg.listeners))
		//}

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

func createUnixSocketListener(param, brigadeID, dir, file string) (net.Listener, error) {
	var path string

	switch param {
	case "-":
		return nil, nil
	case "":
		dir = filepath.Join(dir, brigadeID)
		// create directory if not exists TODO: what permissions do we need?
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
		path = filepath.Join(dir, file)
	default:
		path = param
	}

	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if info != nil {
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory", path)
		}

		if err = os.Remove(path); err != nil {
			return nil, fmt.Errorf("cannot remove socket: %w", err)
		}
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	if err = os.Chmod(path, 0o660); err != nil {
		return nil, fmt.Errorf("chmod 0o660 %s: %w", path, err)
	}

	return l, nil
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

	if *flags.proto0Cfgs != "" {
		vpnCfgs.AddProto0(*flags.proto0Cfgs)
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
