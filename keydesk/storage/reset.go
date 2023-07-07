package storage

import "fmt"

func (db *BrigadeStorage) DomainSet(domain string) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	data.EndpointDomain = domain

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (db *BrigadeStorage) PortSet(port uint16) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	data.EndpointPort = port

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
