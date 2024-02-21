package main

import (
	"errors"
	"net"
	"os"
	"testing"
)

var f flags

func TestMain(m *testing.M) {
	f = parseFlags()
	os.Exit(m.Run())
}

func TestParseArgsComparison(t *testing.T) {
	// TODO: дописать тест, перебрать аргументы

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
	if addrPort != newConfig.addrPort {
		t.Errorf("addrPort mismatch, original: %s, new: %s", addrPort, newConfig.addrPort)
	}
	if id != newConfig.id {
		t.Errorf("id mismatch, original: %s, new: %s", id, newConfig.id)
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
