package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"os"

	gojwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/ssh"
)

var ErrInvalidKey = fmt.Errorf("invalid key type")

func ReadPrivateSSHKey(filename string) (gojwt.SigningMethod, crypto.PrivateKey, crypto.PublicKey, string, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("read private key file %s: %w", filename, err)
	}

	key, err := ssh.ParseRawPrivateKey(buf)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("parse private key: %w", err)
	}

	var (
		jwtMethod gojwt.SigningMethod
		pkey      crypto.PublicKey
	)

	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		jwtMethod = gojwt.SigningMethodES256
		pkey = k.Public()
	case *rsa.PrivateKey:
		jwtMethod = gojwt.SigningMethodRS256
		pkey = k.Public()
	case *ed25519.PrivateKey:
		jwtMethod = gojwt.SigningMethodEdDSA
		pkey = k.Public()
	default:
		return nil, nil, nil, "", fmt.Errorf("unsupported private key type %T", key)
	}

	return jwtMethod, key, pkey, sshKeyId(pkey), nil
}

func ReadPublicSSHKey(filename string) (gojwt.SigningMethod, crypto.PublicKey, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("read private key file %s: %w", filename, err)
	}

	pk, _, _, _, err := ssh.ParseAuthorizedKey(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("parse public key: %w", err)
	}

	ck, ok := pk.(ssh.CryptoPublicKey)
	if !ok {
		return nil, nil, ErrInvalidKey
	}

	pck := ck.CryptoPublicKey()

	switch pk.Type() {
	case "ecdsa-sha2-nistp256":
		return gojwt.SigningMethodES256, pck, nil
	case "ecdsa-sha2-nistp384":
		return gojwt.SigningMethodES384, pck, nil
	case "ecdsa-sha2-nistp521":
		return gojwt.SigningMethodES512, pck, nil
	case "ssh-rsa":
		jwtMethod := gojwt.SigningMethodRS256
		return jwtMethod, pck, nil
	case "ssh-ed25519":
		return gojwt.SigningMethodEdDSA, pck, nil
	default:
		return nil, nil, fmt.Errorf("unsupported ssh public key type: %s", pk.Type())
	}
}

func sshKeyId(pkey crypto.PublicKey) string {
	keyId := "unknown"

	sshpkey, err := ssh.NewPublicKey(pkey)
	if err != nil {
		return keyId
	}

	if sshpkey == nil {
		return keyId
	}

	buf := sha256.Sum256(sshpkey.Marshal())
	if len(buf) < 5 {
		return keyId
	}

	keyId = fmt.Sprintf("%0x", buf[:5])

	return keyId
}
