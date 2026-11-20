package storage

import (
	"fmt"
	"strings"
)

// Country codes the keydesk web has a look for. The default look is "ru" and is
// stored as an empty string, so existing brigade.json files need no migration.
const (
	DefaultCountry = "ru"
	CountryKG      = "kg"
)

var knownCountries = map[string]struct{}{
	DefaultCountry: {},
	CountryKG:      {},
}

// NormalizeCountry - lower-cases a country code and checks that it is one the
// keydesk web knows; an empty code means the default.
func NormalizeCountry(code string) (string, error) {
	country := strings.ToLower(strings.TrimSpace(code))
	if country == "" {
		return DefaultCountry, nil
	}

	if _, ok := knownCountries[country]; !ok {
		return "", fmt.Errorf("unknown country %q", code)
	}

	return country, nil
}

// SetCountry - store the brigade's country (mirror of SetVIP / SetPRO).
// The default country is stored as an empty string.
func (db *BrigadeStorage) SetCountry(code string) error {
	country, err := NormalizeCountry(code)
	if err != nil {
		return err
	}

	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	if country == DefaultCountry {
		brigade.Country = ""
	} else {
		brigade.Country = country
	}

	if err := f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// Country - the brigade's country code, DefaultCountry when unset or unreadable.
func (db *BrigadeStorage) Country() string {
	f, brigade, err := db.openWithReading()
	if err != nil {
		return DefaultCountry
	}
	defer f.Close()

	if brigade == nil || brigade.Country == "" {
		return DefaultCountry
	}

	return brigade.Country
}
