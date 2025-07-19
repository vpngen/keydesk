package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"os"

	gojwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/ssh"
)

func JWTReadPrivateSSHKey(filename string) (gojwt.SigningMethod, crypto.PrivateKey, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("read private key file %s: %w", filename, err)
	}

	key, err := ssh.ParseRawPrivateKey(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key: %w", err)
	}

	var jwtMethod gojwt.SigningMethod

	switch key.(type) {
	case *ecdsa.PrivateKey:
		jwtMethod = gojwt.SigningMethodES256
	case *rsa.PrivateKey:
		jwtMethod = gojwt.SigningMethodRS256
	case *ed25519.PrivateKey:
		jwtMethod = gojwt.SigningMethodEdDSA
	default:
		return nil, nil, fmt.Errorf("unsupported private key type %T", key)
	}

	return jwtMethod, key, nil
}
