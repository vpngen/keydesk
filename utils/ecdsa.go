package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
)

func GenEC256Key() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func WriteECPrivateKey(key *ecdsa.PrivateKey, writer io.Writer) error {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	return pem.Encode(writer, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: b,
	})
}

func ReadECPrivateKey(reader io.Reader) (*ecdsa.PrivateKey, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func EncodePublicKey(key ecdsa.PublicKey) string {
	return base64.StdEncoding.EncodeToString(append(key.X.Bytes(), key.Y.Bytes()...))
}

func DecodePublicKey(encoded string) (ecdsa.PublicKey, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ecdsa.PublicKey{}, err
	}
	return ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(decodedBytes[:32]),
		Y:     new(big.Int).SetBytes(decodedBytes[32:]),
	}, nil
}

func WriteECPublicKey(pub *ecdsa.PublicKey, writer io.Writer) error {
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	return pem.Encode(writer, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	})
}

func ReadECPublicKey(reader io.Reader) (*ecdsa.PublicKey, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%T is not ECDSA public key", key)
	}
	return pub, nil
}
