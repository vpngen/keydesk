package utils

import (
	"bytes"
	"testing"
)

func TestECDSA(t *testing.T) {
	key, err := GenEC256Key()
	if err != nil {
		t.Fatal(err)
	}
	encodedKey := new(bytes.Buffer)
	if err = WriteECPrivateKey(key, encodedKey); err != nil {
		t.Fatal(err)
	}
	decodedKey, err := ReadECPrivateKey(bytes.NewReader(encodedKey.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !key.Equal(decodedKey) {
		t.Fatal("keys are not equal")
	}

	encodedPub := new(bytes.Buffer)
	if err = WriteECPublicKey(&key.PublicKey, encodedPub); err != nil {
		t.Fatal(err)
	}
	decodedPub, err := ReadECPublicKey(bytes.NewReader(encodedPub.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !key.PublicKey.Equal(decodedPub) {
		t.Fatal("public keys are not equal")
	}
}
