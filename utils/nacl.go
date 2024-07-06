package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

type NaCl struct {
	Router, Shuffler [naclkey.NaclBoxKeyLength]byte
}

func (n NaCl) Seal(data []byte) (NaClBox, error) {
	router, err := box.SealAnonymous(nil, data, &n.Router, rand.Reader)
	if err != nil {
		return NaClBox{}, fmt.Errorf("router: %w", err)
	}

	shuffler, err := box.SealAnonymous(nil, data, &n.Shuffler, rand.Reader)
	if err != nil {
		return NaClBox{}, fmt.Errorf("shuffler: %w", err)
	}

	return NaClBox{
		Router:   router,
		Shuffler: shuffler,
	}, nil
}

type Bytes []byte

func (b Bytes) Base64() string {
	return base64.StdEncoding.EncodeToString(b)
}

type NaClBox struct {
	Router, Shuffler Bytes
}
