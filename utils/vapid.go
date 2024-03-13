package utils

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
)

// GenerateVAPIDKeys will create a private and public VAPID key pair
func GenerateVAPIDKeys() (privateKey, publicKey string, err error) {
	private, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return
	}
	// Convert to base64
	publicKey = base64.RawURLEncoding.EncodeToString(private.PublicKey().Bytes())
	privateKey = base64.RawURLEncoding.EncodeToString(private.Bytes())

	return
}
