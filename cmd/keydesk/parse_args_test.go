package main

import (
	"fmt"
	"net"
)

var testArgs = [][]string{
	{"-id", "MRYVCFLQORAINIVIADUTH4OS5Q", "-c", "./", "-w", "./dist", "-a", "-", "-l", "127.0.0.1:80"},
	{"-id", "MRYVCFLQORAINIVIADUTH4OS5Q", "-c", "./", "-w", "./dist", "-a", "-", "-l", "127.0.0.1:80", "-m", "message.sock"},
	{"-id", "MRYVCFLQORAINIVIADUTH4OS5Q", "-c", "./", "-w", "./dist", "-a", "-", "-m", "message.sock"},
}

//func TestParseArgsManual(t *testing.T) {
//	for _, args := range testArgs {
//		f := parseFlags(flag.NewFlagSet(os.Args[0], flag.ExitOnError), args)
//		testParseArgs(f, t)
//	}
//}

func generateCombinations[T any](flags []T) [][]T {
	n := len(flags)
	combinations := make([][]T, 1<<uint(n))
	for i := range combinations {
		comb := make([]T, 0, n)
		for j := 0; j < n; j++ {
			if i&(1<<uint(j)) > 0 {
				comb = append(comb, flags[j])
			}
		}
		combinations[i] = comb
	}
	return combinations
}

type testFlag struct {
	flag   string
	values []string
}

var testFlags = []testFlag{
	{"-c", []string{"etcDir"}},
	{"-w", []string{"webDir"}},
	{"-d", []string{"dbDir"}},
	{"-e", []string{"certDir"}},
	{"-s", []string{"statsDir"}},

	{"-cors", nil},
	{"-id", []string{"MRYVCFLQORAINIVIADUTH4OS5Q"}},
	{"-l", []string{"127.0.0.1:80", "127.0.0.1:443", "127.0.0.1:80,127.0.0.1:443"}},

	//{"-name", []string{"invalidBrigadierName"}},
	//{"-person", []string{"invalidPersonName"}},
	//{"-desc", []string{"invalidPersonDesc"}},
	//{"-url", []string{"invalidPersonURL"}},
	{"-r", nil},

	{"-a", []string{"127.0.0.1:8000", "-"}},

	{"-ch", nil},
	{"-j", nil},

	//{"-wg", []string{"wgcCfgs"}},
	//{"-ovc", []string{"ovcCfgs"}},
	//{"-ipsec", []string{"ipsecCfgs"}},
	//{"-outline", []string{"outlineCfgs"}},
}

//func TestCombineFlags(t *testing.T) {
//	flagCombinations := generateCombinations(testFlags)
//	t.Log(len(flagCombinations))
//	for _, comb := range flagCombinations {
//		recursiveCombine(comb, nil, t)
//	}
//}

//func recursiveCombine(flags []testFlag, results []string, t *testing.T) {
//	if len(flags) == 0 {
//		//fmt.Println(results)
//		t.Run(strings.Join(results, " "), func(t *testing.T) {
//			testParseArgs(parseFlags(flag.NewFlagSet(os.Args[0], flag.ExitOnError), results), t)
//		})
//		return
//	}
//
//	f := flags[0]
//	switch len(f.values) {
//	case 0:
//		recursiveCombine(flags[1:], append(results, f.flag), t)
//	case 1:
//		recursiveCombine(flags[1:], append(results, f.flag, f.values[0]), t)
//	default:
//		for _, v := range f.values {
//			recursiveCombine(flags[1:], append(results, f.flag, v), t)
//		}
//	}
//}

//func testParseArgs(f flags, t *testing.T) {
//	// Call the original function
//	chunked, jsonOut, enableCORS, listeners, addrPort, id, etcDir, webDir, dbDir, certDir, statsDir, brigadierName, person, replaceBrigadier, _, err := parseArgs(f)
//
//	// Call the new function
//	newConfig, newErr := parseArgs2(f)
//
//	if err != nil && newErr != nil {
//		if err.Error() != newErr.Error() {
//			t.Fatalf("error mismatch, original: %s, new: %s", err, newErr)
//		}
//		return
//	}
//
//	// Compare configurations
//	if chunked != newConfig.chunked {
//		t.Fatalf("chunked mismatch, original: %t, new: %t", chunked, newConfig.chunked)
//	}
//	if jsonOut != newConfig.jsonOut {
//		t.Fatalf("jsonOut mismatch, original: %t, new: %t", jsonOut, newConfig.jsonOut)
//	}
//	if enableCORS != newConfig.enableCORS {
//		t.Fatalf("enableCORS mismatch, original: %t, new: %t", enableCORS, newConfig.enableCORS)
//	}
//	if !listenersEqual(listeners, newConfig.listeners) {
//		t.Error("listeners mismatch")
//	}
//	if addrPort != newConfig.addr {
//		t.Fatalf("addr mismatch, original: %s, new: %s", addrPort, newConfig.addr)
//	}
//	if id != newConfig.brigadeID {
//		t.Fatalf("brigadeID mismatch, original: %s, new: %s", id, newConfig.brigadeID)
//	}
//	if etcDir != newConfig.etcDir {
//		t.Fatalf("etcDir mismatch, original: %s, new: %s", etcDir, newConfig.etcDir)
//	}
//	if webDir != newConfig.webDir {
//		t.Fatalf("webDir mismatch, original: %s, new: %s", webDir, newConfig.webDir)
//	}
//	if dbDir != newConfig.dbDir {
//		t.Fatalf("dbDir mismatch, original: %s, new: %s", dbDir, newConfig.dbDir)
//	}
//	if certDir != newConfig.certDir {
//		t.Fatalf("certDir mismatch, original: %s, new: %s", certDir, newConfig.certDir)
//	}
//	if statsDir != newConfig.statsDir {
//		t.Fatalf("statsDir mismatch, original: %s, new: %s", statsDir, newConfig.statsDir)
//	}
//	if brigadierName != newConfig.brigadierName {
//		t.Fatalf("brigadierName mismatch, original: %s, new: %s", brigadierName, newConfig.brigadierName)
//	}
//	if person != newConfig.person {
//		t.Error("person mismatch")
//	}
//	if replaceBrigadier != newConfig.replaceBrigadier {
//		t.Fatalf("replaceBrigadier mismatch, original: %t, new: %t", replaceBrigadier, newConfig.replaceBrigadier)
//	}
//}

func listenersEqual(a, b []net.Listener) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Addr() != b[i].Addr() {
			fmt.Println(a[i].Addr(), b[i].Addr())
			return false
		}
	}
	return true
}

//func parseArgs(flags flags) (bool, bool, bool, []net.Listener, netip.AddrPort, string, string, string, string, string, string, string, namesgenerator.Person, bool, *storage.ConfigsImplemented, error) {
//	var (
//		id                               string
//		etcdir, dbdir, certdir, statsdir string
//		person                           namesgenerator.Person
//		addrPort                         netip.AddrPort
//	)
//
//	sysUser, err := user.Current()
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot define user: %w", err)
//	}
//
//	vpnCfgs := storage.NewConfigsImplemented()
//
//	if *flags.wgcCfgs != "" {
//		vpnCfgs.AddWg(*flags.wgcCfgs)
//	}
//
//	if *flags.ovcCfgs != "" {
//		vpnCfgs.AddOvc(*flags.ovcCfgs)
//	}
//
//	if *flags.ipsecCfgs != "" {
//		vpnCfgs.AddIPSec(*flags.ipsecCfgs)
//	}
//
//	if *flags.outlineCfgs != "" {
//		vpnCfgs.AddOutline(*flags.outlineCfgs)
//	}
//
//	if *flags.webDir == "" {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrStaticDirEmpty
//	}
//
//	webdir, err := filepath.Abs(*flags.webDir)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("web dir: %w", err)
//	}
//
//	if *flags.filedbDir != "" {
//		dbdir, err = filepath.Abs(*flags.filedbDir)
//		if err != nil {
//			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("dbdir dir: %w", err)
//		}
//	}
//
//	if *flags.etcDir != "" {
//		etcdir, err = filepath.Abs(*flags.etcDir)
//		if err != nil {
//			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("etcdir dir: %w", err)
//		}
//	}
//
//	if *flags.certDir != "" {
//		certdir, err = filepath.Abs(*flags.certDir)
//		if err != nil {
//			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("certdir dir: %w", err)
//		}
//	}
//
//	if *flags.statsDir != "" {
//		statsdir, err = filepath.Abs(*flags.statsDir)
//		if err != nil {
//			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("statdir dir: %w", err)
//		}
//	}
//
//	switch *flags.brigadeID {
//	case "", sysUser.Username:
//		id = sysUser.Username
//
//		if *flags.filedbDir == "" {
//			dbdir = filepath.Join(storage.DefaultHomeDir, id)
//		}
//
//		if *flags.etcDir == "" {
//			etcdir = keydesk.DefaultEtcDir
//		}
//
//		if *flags.certDir == "" {
//			certdir = DefaultCertDir
//		}
//
//		if *flags.statsDir == "" {
//			statsdir = filepath.Join(storage.DefaultStatsDir, id)
//		}
//	default:
//		id = *flags.brigadeID
//
//		cwd, err := os.Getwd()
//		if err == nil {
//			cwd, _ = filepath.Abs(cwd)
//		}
//
//		if *flags.filedbDir == "" {
//			dbdir = cwd
//		}
//
//		if *flags.etcDir == "" {
//			etcdir = cwd
//		}
//
//		if *flags.certDir == "" {
//			certdir = cwd
//		}
//
//		if *flags.statsDir == "" {
//			statsdir = cwd
//		}
//	}
//
//	// brigadeID must be base32 decodable.
//	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("id base32: %s: %w", id, err)
//	}
//
//	_, err = uuid.FromBytes(binID)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("id uuid: %s: %w", id, err)
//	}
//
//	if *flags.addr != "-" {
//		addrPort, err = netip.ParseAddrPort(*flags.addr)
//		if err != nil {
//			return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("api addr: %w", err)
//		}
//	}
//
//	if *flags.replaceBrigadier {
//		return *flags.chunked, *flags.jsonOut, *flags.pcors, nil, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, "", person, *flags.replaceBrigadier, vpnCfgs, nil
//	}
//
//	_, err = parseMessageAPISocket(flags, config{brigadeID: id})
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, err
//	}
//
//	if *flags.brigadierName == "" {
//		var listeners []net.Listener
//		// get listeners from argument
//		for _, laddr := range strings.Split(*flags.listenAddr, ",") {
//			if laddr == "" {
//				continue
//			}
//			l, err := net.Listen("tcp", laddr)
//			if err != nil {
//				return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot listen: %w", err)
//			}
//
//			listeners = append(listeners, l)
//		}
//
//		//if len(listeners) != 1 && len(listeners) != 2 {
//		//	return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("unexpected number of litening (%d != 1|2)",
//		//		len(listeners))
//		//}
//
//		return *flags.chunked, *flags.jsonOut, *flags.pcors, listeners, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, "", person, false, nil, nil
//	}
//
//	// brigadierName must be not empty and must be a valid UTF8 string
//	buf, err := base64.StdEncoding.DecodeString(*flags.brigadierName)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("brigadier name: %w", err)
//	}
//
//	if !utf8.Valid(buf) {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidBrigadierName
//	}
//
//	name := string(buf)
//
//	// personName must be not empty and must be a valid UTF8 string
//	if *flags.personName == "" {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrEmptyPersonName
//	}
//
//	buf, err = base64.StdEncoding.DecodeString(*flags.personName)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("person name: %w", err)
//	}
//
//	if !utf8.Valid(buf) {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidPersonName
//	}
//
//	person.Name = string(buf)
//
//	// personDesc must be not empty and must be a valid UTF8 string
//	if *flags.personDesc == "" {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrEmptyPersonDesc
//	}
//
//	buf, err = base64.StdEncoding.DecodeString(*flags.personDesc)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("person desc: %w", err)
//	}
//
//	if !utf8.Valid(buf) {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidPersonDesc
//	}
//
//	person.Desc = string(buf)
//
//	// personURL must be not empty and must be a valid UTF8 string
//	if *flags.personURL == "" {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrEmptyPersonURL
//	}
//
//	buf, err = base64.StdEncoding.DecodeString(*flags.personURL)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("person url: %w", err)
//	}
//
//	if !utf8.Valid(buf) {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, ErrInvalidPersonURL
//	}
//
//	u := string(buf)
//
//	_, err = url.Parse(u)
//	if err != nil {
//		return false, false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, nil, fmt.Errorf("parse person url: %w", err)
//	}
//
//	person.URL = u
//
//	return *flags.chunked, *flags.jsonOut, *flags.pcors, nil, addrPort, id, etcdir, webdir, dbdir, certdir, statsdir, name, person, *flags.replaceBrigadier, vpnCfgs, nil
//}
