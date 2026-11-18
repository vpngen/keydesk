package storage

import "testing"

func TestCountryDefault(t *testing.T) {
	if got := db.Country(); got != DefaultCountry {
		t.Fatalf("fresh brigade country = %q, want %q", got, DefaultCountry)
	}
}

func TestSetCountry(t *testing.T) {
	if err := db.SetCountry("KG"); err != nil {
		t.Fatalf("set country kg: %s", err)
	}
	if got := db.Country(); got != CountryKG {
		t.Fatalf("country after set = %q, want %q", got, CountryKG)
	}

	// The default is stored as an empty string, whichever way it is spelled.
	for _, code := range []string{"ru", " RU ", ""} {
		if err := db.SetCountry(code); err != nil {
			t.Fatalf("set country %q: %s", code, err)
		}
		if got := db.Country(); got != DefaultCountry {
			t.Fatalf("country after set %q = %q, want %q", code, got, DefaultCountry)
		}
	}

	if err := db.SetCountry("xx"); err == nil {
		t.Fatal("unknown country accepted")
	}
	if got := db.Country(); got != DefaultCountry {
		t.Fatalf("country changed by a rejected code: %q", got)
	}
}
