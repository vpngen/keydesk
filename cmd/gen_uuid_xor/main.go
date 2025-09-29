package main

import (
	"encoding/base32"
	"fmt"
	"os"

	"github.com/google/uuid"
)

func deriveUUIDv4() {
	u := uuid.New()

	u[6] &= 0x0F
	u[8] &= 0x3F

	fmt.Println(u)
}

func xorUUIDs(secret, base32uuid string) error {
	s, err := uuid.Parse(secret)
	if err != nil {
		return fmt.Errorf("parsing secret uuid: %w", err)
	}

	b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(base32uuid)
	if err != nil {
		return fmt.Errorf("decoding base32 uuid: %w", err)
	}

	if len(b) != 16 {
		return fmt.Errorf("decoded base32 uuid is not 16 bytes")
	}

	var r uuid.UUID
	for i := range 16 {
		r[i] = s[i] ^ b[i]
	}

	fmt.Println(r)

	return nil
}

func main() {
	switch len(os.Args) {
	case 1:
		deriveUUIDv4()
	case 3:
		if err := xorUUIDs(os.Args[1], os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Usage: %s [<uuid secret> <based32 uuid>]\n", os.Args[0])
		os.Exit(1)
	}
}
