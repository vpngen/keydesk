package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"math/big"
)

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
