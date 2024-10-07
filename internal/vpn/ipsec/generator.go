package ipsec

import (
	"crypto/rand"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
)

func Generate(brigade *storage.Brigade, user *storage.User, nacl utils.NaCl, epData map[string]string) (ClientConfig, error) {
	usernameRand := make([]byte, UsernameLen)
	if _, err := rand.Read(usernameRand); err != nil {
		return ClientConfig{}, fmt.Errorf("username rand: %w", err)
	}

	passwordRand := make([]byte, PasswordLen)
	if _, err := rand.Read(passwordRand); err != nil {
		return ClientConfig{}, fmt.Errorf("password rand: %w", err)
	}

	username := base58.Encode(usernameRand)[:UsernameLen]
	password := base58.Encode(passwordRand)[:UsernameLen]

	encUser, err := nacl.Seal([]byte(username))
	if err != nil {
		return ClientConfig{}, fmt.Errorf("username seal: %w", err)
	}
	encPass, err := nacl.Seal([]byte(password))
	if err != nil {
		return ClientConfig{}, fmt.Errorf("password seal: %w", err)
	}

	epData["l2tp-username"] = encUser.Router.Base64()
	epData["l2tp-password"] = encPass.Router.Base64()

	user.IPSecUsernameRouterEnc = encUser.Router.Base64()
	user.IPSecUsernameShufflerEnc = encUser.Shuffler.Base64()
	user.IPSecPasswordRouterEnc = encPass.Router.Base64()
	user.IPSecPasswordShufflerEnc = encPass.Shuffler.Base64()

	return ClientConfig{
		Username: username,
		Password: password,
		Host:     storage.GetEndpointHost(brigade, user),
		PSK:      brigade.IPSecPSK,
	}, nil
}
