package openvpn

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
)

func csrPemGzBase64(csr []byte) ([]byte, error) {
	return kdlib.PemGzipBase64(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
}

func Generate(brigade *storage.Brigade, user *storage.User, nacl utils.NaCl, epData map[string]string) (Config, error) {
	cn := uuid.New()
	csr, key, err := kdlib.NewOvClientCertRequest(cn.String())
	if err != nil {
		return Config{}, fmt.Errorf("openvpn csr: %w", err)
	}

	csrEnc, err := csrPemGzBase64(csr)
	if err != nil {
		return Config{}, fmt.Errorf("csr encode: %w", err)
	}
	user.OvCSRGzipBase64 = string(csrEnc)
	epData["openvpn-client-csr"] = string(csrEnc)

	keyPem, err := keyPEM(key)
	if err != nil {
		return Config{}, fmt.Errorf("encode key pem: %w", err)
	}

	caPem, err := kdlib.Unbase64Ungzip(brigade.OvCACertPemGzipBase64)
	if err != nil {
		return Config{}, fmt.Errorf("ca decode: %w", err)
	}

	return Config{
		DNS: "100.126.0.1",
		IP:  brigade.EndpointIPv4.String(),
		CA:  string(caPem),
		Key: string(keyPem),
	}, nil
}

func keyPEM(k *ecdsa.PrivateKey) ([]byte, error) {
	key, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return nil, fmt.Errorf("key marshal: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), nil
}
