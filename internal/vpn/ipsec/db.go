package ipsec

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
)

type EncryptedConfig struct {
	routerUser, routerPass, shufflerUser, shufflerPass string
}

func (c Config) SaveToUser(user *storage.User, router, shuffler [32]byte) error {
	ec, err := c.encrypt(router, shuffler)
	if err != nil {
		return err
	}
	user.IPSecUsernameRouterEnc = ec.routerUser
	user.IPSecUsernameShufflerEnc = ec.shufflerUser
	user.IPSecPasswordRouterEnc = ec.routerPass
	user.IPSecPasswordShufflerEnc = ec.shufflerPass
	return nil
}

func (c Config) encrypt(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (EncryptedConfig, error) {
	// TODO: why do we encrypt *encoded* secret?
	routerUser, err := box.SealAnonymous(nil, []byte(c.username), &routerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("username router seal: %w", err)
	}

	routerPass, err := box.SealAnonymous(nil, []byte(c.password), &routerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("password router seal: %w", err)
	}

	shufflerUser, err := box.SealAnonymous(nil, []byte(c.username), &shufflerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("username shuffler seal: %w", err)
	}

	shufflerPass, err := box.SealAnonymous(nil, []byte(c.password), &shufflerPub, rand.Reader)
	if err != nil {
		return EncryptedConfig{}, fmt.Errorf("password shuffler seal: %w", err)
	}

	return EncryptedConfig{
		routerUser:   base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(routerUser),
		routerPass:   base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(routerPass),
		shufflerUser: base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(shufflerUser),
		shufflerPass: base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(shufflerPass),
	}, nil
}
