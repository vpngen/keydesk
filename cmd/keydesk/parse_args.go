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

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	jwtsvc "github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/utils"
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

	unixSocketDir             *string
	messageAPI                *string
	shufflerAPI               *string
	msgJwtPubkeyFilename      *string
	keydeskJwtPrivkeyFilename *string
}

const (
	defaultUnixSocketDir      = "/var/lib/dcapi"
	defaultMessageSocket      = "messages.sock"
	defaultShufflerSocket     = "shuffler.sock"
	jwtPubKeyFileName         = "jwt-pub-msg.pem"
	msgJwtPubkeyFilename      = "msg-jwt.pub"
	keydeskJwtPrivkeyFileName = "keydesk-jwt.key"
	etcSubdir                 = "vg-keydesk"
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
	f.msgJwtPubkeyFilename = flagSet.String("msgjwt", "", fmt.Sprintf("Path to Messages JWT public key file. Default: %s/%s", keydesk.DefaultEtcDir, jwtPubKeyFileName))
	f.keydeskJwtPrivkeyFilename = flagSet.String("kdjwt", "", fmt.Sprintf("Path to Keydesk JWT private key file. Default: %s/%s", keydesk.DefaultEtcDir, keydeskJwtPrivkeyFileName))

	// ignore errors, see original flag.Parse() func
	_ = flagSet.Parse(args)

	return f
}

type config struct {
	chunked             bool
	jsonOut             bool
	enableCORS          bool
	listeners           []net.Listener
	addr                netip.AddrPort
	brigadeID           string
	brigadeUUIDofbs     string
	etcDir              string
	webDir              string
	dbDir               string
	certDir             string
	statsDir            string
	brigadierName       string
	person              namesgenerator.Person
	replaceBrigadier    bool
	vpnConfigs          *storage.ConfigsImplemented
	unixSocketDir       string
	messageAPISocket    net.Listener
	shufflerAPISocket   net.Listener
	jwtKeydeskIssuer    jwtsvc.KeydeskTokenIssuer
	jwtKeydesAuthorizer jwtsvc.KeydeskTokenAuthorizer
	jwtMsgAuthorizer    jwtsvc.MessagesJwtAuthorizer
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

	brigadeUUID, err := checkBase32EncodedUUID(cfg.brigadeID)
	if err != nil {
		return cfg, err
	}

	obfsKey := os.Getenv("OBFS_UUID")
	fmt.Fprintf(os.Stderr, "obfs uuid: %s\n", obfsKey)
	obfsUUID, err := uuid.Parse(obfsKey)
	if err == nil {
		fmt.Fprintf(os.Stderr, "obfs uuid parsed: %s\n", obfsUUID)
		var brigadeUUIDofbs uuid.UUID
		for i := range 16 {
			brigadeUUIDofbs[i] = brigadeUUID[i] ^ obfsUUID[i]
		}

		fmt.Fprintf(os.Stderr, "obfs brigade uuid: %s\n", brigadeUUIDofbs)

		cfg.brigadeUUIDofbs = brigadeUUIDofbs.String()
	}

	if cfg.brigadeUUIDofbs == "" {
		cfg.brigadeUUIDofbs = uuid.Nil.String()
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
		if *flags.messageAPI != "-" {
			opts := jwtsvc.MessagesJwtOptions{
				Issuer:   "dc-mgmt",
				Audience: []string{"keydesk"},
			}

			fn := *flags.msgJwtPubkeyFilename
			if fn == "" {
				fn = filepath.Join(cfg.etcDir, jwtPubKeyFileName)
				if _, err := os.Stat(fn); !os.IsNotExist(err) {
					if file, err := os.Open(fn); err == nil {
						if key, err := utils.ReadECPublicKey(file); err == nil {
							opts.SigningMethod = jwt.SigningMethodES256
							cfg.jwtMsgAuthorizer = jwtsvc.NewMessagesJwtAuthorizer(key, opts)
						}
					}
				}

				fn = filepath.Join(cfg.etcDir, msgJwtPubkeyFilename)
				if _, err := os.Stat(fn); !os.IsNotExist(err) {
					if _, err := os.Stat(fn); !os.IsNotExist(err) {
						method, key, err := jwtsvc.ReadPublicSSHKey(fn)
						if err == nil {
							opts.SigningMethod = method
							cfg.jwtMsgAuthorizer = jwtsvc.NewMessagesJwtAuthorizer(key, opts)
						}
					}
				}
			}

			if cfg.jwtMsgAuthorizer.IsNil() {
				return cfg, fmt.Errorf("cannot read jwt messages public key from %s", fn)
			}
		}

		vipPrivkeyFn := *flags.keydeskJwtPrivkeyFilename
		if vipPrivkeyFn == "" {
			vipPrivkeyFn = filepath.Join(cfg.etcDir, etcSubdir, keydeskJwtPrivkeyFileName)
		}

		_, err := os.Stat(vipPrivkeyFn)
		exists := !os.IsNotExist(err)

		switch {
		case *flags.keydeskJwtPrivkeyFilename == "" && !exists:
			secret, err := utils.GenHMACKey()
			if err != nil {
				return cfg, fmt.Errorf("generate jwt vip secret: %w", err)
			}

			jwtopts := jwtsvc.KeydeskTokenOptions{
				Issuer:        "keydesk",
				Subject:       cfg.brigadeUUIDofbs,
				Audience:      []string{"keydesk"},
				SigningMethod: jwt.SigningMethodHS256,
			}

			cfg.jwtKeydesAuthorizer = jwtsvc.NewKeydeskTokenAuthorizer(secret, jwtopts)

			jwtopts.Audience = append(jwtopts.Audience, "socket")
			cfg.jwtKeydeskIssuer = jwtsvc.NewKeydeskTokenIssuer(secret, "random", jwtopts)
		case exists:
			signingMethod, jwtKeydeskPrivkey, jwtKeydeskPubkey, keyId, err := jwtsvc.ReadPrivateSSHKey(vipPrivkeyFn)
			if err != nil {
				return cfg, fmt.Errorf("read jwt vip private key: %w", err)
			}

			jwtopts := jwtsvc.KeydeskTokenOptions{
				Issuer:        "keydesk",
				Subject:       cfg.brigadeUUIDofbs,
				Audience:      []string{"keydesk"},
				SigningMethod: signingMethod,
			}

			cfg.jwtKeydesAuthorizer = jwtsvc.NewKeydeskTokenAuthorizer(jwtKeydeskPubkey, jwtopts)

			jwtopts.Audience = append(jwtopts.Audience, "socket")
			cfg.jwtKeydeskIssuer = jwtsvc.NewKeydeskTokenIssuer(jwtKeydeskPrivkey, keyId, jwtopts)
		default:
			return cfg, fmt.Errorf("jwt vip private key file %s set but does not exist", vipPrivkeyFn)
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

func checkBase32EncodedUUID(s string) (uuid.UUID, error) {
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("id base32: %s: %w", s, err)
	}

	u, err := uuid.FromBytes(binID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("id uuid: %s: %w", s, err)
	}

	return u, nil
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
