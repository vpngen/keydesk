package keydesk

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// CreateBrigade - create brigadier user.
func CreateBrigade(
	db *storage.BrigadeStorage,
	vpnCfgs *storage.ConfigsImplemented,
	config *storage.BrigadeConfig,
	routerPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte,
	mode storage.Mode,
	maxUsers uint,
) error {
	wgConf, err := genEndpointWGKeys(routerPubkey, shufflerPubkey)
	if err != nil {
		return fmt.Errorf("wg keys: %w", err)
	}

	// fmt.Fprintf(os.Stderr, "cfgs: %#v\n", vpnCfgs)

	var ovcConf *storage.BrigadeOvcConfig
	if len(vpnCfgs.Ovc) > 0 {
		var err error

		ovcConf, err = GenEndpointOpenVPNoverCloakCreds(routerPubkey, shufflerPubkey)
		if err != nil {
			return fmt.Errorf("ovc creds: %w", err)
		}
	}

	var ipsecConf *storage.BrigadeIPSecConfig
	if len(vpnCfgs.IPSec) > 0 {
		ipsecConf, err = GenEndpointIPSecCreds(routerPubkey, shufflerPubkey)
		if err != nil {
			return fmt.Errorf("ipsec psk: %w", err)
		}
	}

	var outlineConf *storage.BrigadeOutlineConfig
	if len(vpnCfgs.Outline) > 0 {
		outlineConf = &storage.BrigadeOutlineConfig{OutlinePort: config.OutlinePort}
	}

	err = db.CreateBrigade(config, wgConf, ovcConf, ipsecConf, outlineConf, mode, maxUsers)
	if err != nil {
		return fmt.Errorf("put: %w", err)
	}

	return nil
}

// DestroyBrigade - destroy brigadier user.
func DestroyBrigade(db *storage.BrigadeStorage) error {
	err := db.DestroyBrigade()
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	return nil
}

func genEndpointWGKeys(routerPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) (*storage.BrigadeWgConfig, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("gen wg key: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, key[:], routerPubkey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, key[:], shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("shuffler seal: %w", err)
	}

	pub := key.PublicKey()

	return &storage.BrigadeWgConfig{
		WgPublicKey:          pub[:],
		WgPrivateRouterEnc:   routerKey,
		WgPrivateShufflerEnc: shufflerKey,
	}, nil
}

func GenEndpointOpenVPNoverCloakCreds(routerPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) (*storage.BrigadeOvcConfig, error) {
	cert, key, err := kdlib.NewOvCA()
	if err != nil {
		return nil, fmt.Errorf("ov new ca: %w", err)
	}

	caPemGzipBase64, err := kdlib.PemGzipBase64(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if err != nil {
		return nil, fmt.Errorf("crt pem encode: %w", err)
	}

	caKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}

	keyPemGz, err := kdlib.PemGzip(&pem.Block{Type: "PRIVATE KEY", Bytes: caKey})
	if err != nil {
		return nil, fmt.Errorf("key pem encode: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, keyPemGz, routerPubkey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, keyPemGz, shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("shuffler seal: %w", err)
	}

	return &storage.BrigadeOvcConfig{
		OvcFakeDomain:          GetRandomSite(),
		OvcCACertPemGzipBase64: string(caPemGzipBase64),
		OvcRouterCAKey:         base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(routerKey),
		OvcShufflerCAKey:       base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(shufflerKey),
	}, nil
}
