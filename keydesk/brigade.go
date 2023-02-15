package keydesk

import (
	"crypto/rand"
	"fmt"

	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// CreateBrigade - create brigadier user.
func CreateBrigade(db *BrigadeStorage, config *BrigadeConfig, routerPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) error {
	wgPub, wgRouterPriv, wgShufflerPriv, err := genEndpointWGKeys(routerPubkey, shufflerPubkey)
	if err != nil {
		return fmt.Errorf("wg keys: %w", err)
	}

	err = db.BrigadePut(config, wgPub, wgRouterPriv, wgShufflerPriv)
	if err != nil {
		fmt.Printf("put: %s", err)

		return fmt.Errorf("put: %w", err)
	}

	return nil
}

// DestroyBrigade - destroy brigadier user.
func DestroyBrigade(db *BrigadeStorage) error {
	err := db.BrigadeRemove()
	if err != nil {
		fmt.Printf("remove: %s", err)

		return fmt.Errorf("remove: %w", err)
	}

	return nil
}

func genEndpointWGKeys(routerPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gen wg key: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, key[:], routerPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, key[:], shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("shuffler seal: %w", err)
	}

	pub := key.PublicKey()

	return pub[:], routerKey, shufflerKey, nil
}
