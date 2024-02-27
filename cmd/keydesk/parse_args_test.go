package main

import (
	"errors"
	"flag"
	"net"
	"os"
	"strings"
	"testing"
)

var testArgs = [][]string{
	{"-id", "MRYVCFLQORAINIVIADUTH4OS5Q", "-c", "./", "-w", "./dist", "-a", "-"},
}

func TestParseArgsManual(t *testing.T) {
	for _, args := range testArgs {
		f := parseFlags(flag.NewFlagSet(os.Args[0], flag.ExitOnError), args)

		// Call the original function
		chunked, jsonOut, enableCORS, listeners, addrPort, id, etcDir, webDir, dbDir, certDir, statsDir, brigadierName, person, replaceBrigadier, _, err := parseArgs(f)

		// Call the new function
		newConfig, newErr := parseArgs2(f)

		// Compare errors
		if !errors.Is(err, newErr) {
			t.Errorf("error mismatch, original: %s, new: %s", err, newErr)
		}

		// Compare configurations
		if chunked != newConfig.chunked {
			t.Errorf("chunked mismatch, original: %t, new: %t", chunked, newConfig.chunked)
		}
		if jsonOut != newConfig.jsonOut {
			t.Errorf("jsonOut mismatch, original: %t, new: %t", jsonOut, newConfig.jsonOut)
		}
		if enableCORS != newConfig.enableCORS {
			t.Errorf("enableCORS mismatch, original: %t, new: %t", enableCORS, newConfig.enableCORS)
		}
		if !listenersEqual(listeners, newConfig.listeners) {
			t.Error("listeners mismatch")
		}
		if addrPort != newConfig.addr {
			t.Errorf("addr mismatch, original: %s, new: %s", addrPort, newConfig.addr)
		}
		if id != newConfig.brigadeID {
			t.Errorf("brigadeID mismatch, original: %s, new: %s", id, newConfig.brigadeID)
		}
		if etcDir != newConfig.etcDir {
			t.Errorf("etcDir mismatch, original: %s, new: %s", etcDir, newConfig.etcDir)
		}
		if webDir != newConfig.webDir {
			t.Errorf("webDir mismatch, original: %s, new: %s", webDir, newConfig.webDir)
		}
		if dbDir != newConfig.dbDir {
			t.Errorf("dbDir mismatch, original: %s, new: %s", dbDir, newConfig.dbDir)
		}
		if certDir != newConfig.certDir {
			t.Errorf("certDir mismatch, original: %s, new: %s", certDir, newConfig.certDir)
		}
		if statsDir != newConfig.statsDir {
			t.Errorf("statsDir mismatch, original: %s, new: %s", statsDir, newConfig.statsDir)
		}
		if brigadierName != newConfig.brigadierName {
			t.Errorf("brigadierName mismatch, original: %s, new: %s", brigadierName, newConfig.brigadierName)
		}
		if person != newConfig.person {
			t.Error("person mismatch")
		}
		if replaceBrigadier != newConfig.replaceBrigadier {
			t.Errorf("replaceBrigadier mismatch, original: %t, new: %t", replaceBrigadier, newConfig.replaceBrigadier)
		}
	}
}

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

func TestCombineFlags(t *testing.T) {
	flagCombinations := generateCombinations(testFlags)
	t.Log(len(flagCombinations))
	for _, comb := range flagCombinations {
		recursiveCombine(comb, nil, t)
	}
}

func recursiveCombine(flags []testFlag, results []string, t *testing.T) {
	if len(flags) == 0 {
		//fmt.Println(results)
		t.Run(strings.Join(results, " "), func(t *testing.T) {
			testParseArgs(parseFlags(flag.NewFlagSet(os.Args[0], flag.ExitOnError), results), t)
		})
		return
	}

	f := flags[0]
	switch len(f.values) {
	case 0:
		recursiveCombine(flags[1:], append(results, f.flag), t)
	case 1:
		recursiveCombine(flags[1:], append(results, f.flag, f.values[0]), t)
	default:
		for _, v := range f.values {
			recursiveCombine(flags[1:], append(results, f.flag, v), t)
		}
	}
}

func testParseArgs(f flags, t *testing.T) {
	// Call the original function
	chunked, jsonOut, enableCORS, listeners, addrPort, id, etcDir, webDir, dbDir, certDir, statsDir, brigadierName, person, replaceBrigadier, _, err := parseArgs(f)

	// Call the new function
	newConfig, newErr := parseArgs2(f)

	if err != nil && newErr != nil {
		if err.Error() != newErr.Error() {
			t.Fatalf("error mismatch, original: %s, new: %s", err, newErr)
		}
		return
	}

	// Compare configurations
	if chunked != newConfig.chunked {
		t.Fatalf("chunked mismatch, original: %t, new: %t", chunked, newConfig.chunked)
	}
	if jsonOut != newConfig.jsonOut {
		t.Fatalf("jsonOut mismatch, original: %t, new: %t", jsonOut, newConfig.jsonOut)
	}
	if enableCORS != newConfig.enableCORS {
		t.Fatalf("enableCORS mismatch, original: %t, new: %t", enableCORS, newConfig.enableCORS)
	}
	if !listenersEqual(listeners, newConfig.listeners) {
		t.Error("listeners mismatch")
	}
	if addrPort != newConfig.addr {
		t.Fatalf("addr mismatch, original: %s, new: %s", addrPort, newConfig.addr)
	}
	if id != newConfig.brigadeID {
		t.Fatalf("brigadeID mismatch, original: %s, new: %s", id, newConfig.brigadeID)
	}
	if etcDir != newConfig.etcDir {
		t.Fatalf("etcDir mismatch, original: %s, new: %s", etcDir, newConfig.etcDir)
	}
	if webDir != newConfig.webDir {
		t.Fatalf("webDir mismatch, original: %s, new: %s", webDir, newConfig.webDir)
	}
	if dbDir != newConfig.dbDir {
		t.Fatalf("dbDir mismatch, original: %s, new: %s", dbDir, newConfig.dbDir)
	}
	if certDir != newConfig.certDir {
		t.Fatalf("certDir mismatch, original: %s, new: %s", certDir, newConfig.certDir)
	}
	if statsDir != newConfig.statsDir {
		t.Fatalf("statsDir mismatch, original: %s, new: %s", statsDir, newConfig.statsDir)
	}
	if brigadierName != newConfig.brigadierName {
		t.Fatalf("brigadierName mismatch, original: %s, new: %s", brigadierName, newConfig.brigadierName)
	}
	if person != newConfig.person {
		t.Error("person mismatch")
	}
	if replaceBrigadier != newConfig.replaceBrigadier {
		t.Fatalf("replaceBrigadier mismatch, original: %t, new: %t", replaceBrigadier, newConfig.replaceBrigadier)
	}
}

func listenersEqual(a, b []net.Listener) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
