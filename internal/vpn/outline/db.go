package outline

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

func (c Config) Store(user *storage.User, router, shuffler [32]byte) error {
	enc, err := c.encrypt(router, shuffler)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	user.OutlineSecretRouterEnc = enc.routerSecret
	user.OutlineSecretShufflerEnc = enc.shufflerSecret
	return nil
}

func (c Config) encrypt(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (EncryptedConfig, error) {
	// TODO: why do we encrypt *encoded* secret?
	secretRouter, err := box.SealAnonymous(nil, []byte(c.secret), &routerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("secret router seal: %w", err)
	}

	secretShuffler, err := box.SealAnonymous(nil, []byte(c.secret), &shufflerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("secret shuffler seal: %w", err)
	}

	return EncryptedConfig{
		routerSecret:   base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretRouter),
		shufflerSecret: base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretShuffler),
	}, nil
}
