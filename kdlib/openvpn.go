package kdlib

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
)

const CAdays = 365 * 20

var ErrCertExiresAfterCA = errors.New("cannot create certificate that expires after the CA")

// x509.KeyUsageCRLSign|x509.KeyUsageCertSign|x509.KeyUsageDigitalSignature
func NewOvCA() ([]byte, *ecdsa.PrivateKey, error) {
	timestamp := time.Now().UTC()
	name := uuid.New().String()

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt8))
	if err != nil {
		return nil, nil, fmt.Errorf("gen serial: %w", err)
	}

	cert := &x509.Certificate{
		IsCA:                  true,
		Subject:               NewOvSubjectName(name),
		Version:               0,
		KeyUsage:              x509.KeyUsageCRLSign | x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		NotAfter:              timestamp.AddDate(0, 0, CAdays),
		NotBefore:             timestamp,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		SerialNumber:          serial,
		PublicKeyAlgorithm:    x509.ECDSA,
		SignatureAlgorithm:    x509.ECDSAWithSHA512,
		BasicConstraintsValid: true,
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("gen key: %w", err)
	}

	raw, err := x509.CreateCertificate(rand.Reader, cert, cert, privKey.Public(), privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create cert: %w", err)
	}

	return raw, privKey, err
}

// x509.KeyUsageDigitalSignature, x509.ExtKeyUsageClientAuth
func NewOvClientCertRequest(name string) ([]byte, *ecdsa.PrivateKey, error) {
	certReq := &x509.CertificateRequest{
		Subject:            NewOvSubjectName(name),
		Version:            0,
		PublicKeyAlgorithm: x509.ECDSA,
		SignatureAlgorithm: x509.ECDSAWithSHA512,
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("gen key: %w", err)
	}

	raw, err := x509.CreateCertificateRequest(rand.Reader, certReq, privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create cerr req: %w", err)
	}

	return raw, privKey, err
}

func NewOvSubjectName(n string) pkix.Name {
	return pkix.Name{
		Country:            []string{},
		Locality:           []string{},
		Province:           []string{},
		PostalCode:         []string{},
		CommonName:         OvParseCommonName(n),
		Organization:       []string{},
		StreetAddress:      []string{},
		OrganizationalUnit: []string{},
	}
}

func OvParseCommonName(s string) string {
	if len(s) == 0 {
		return s
	}

	r := make([]byte, 0, len(s))

	for i := range s {
		if s[i] < 45 || s[i] >= 127 {
			continue
		}

		switch s[i] {
		case '/', '@', '`', '[', ']', '\\', '^', ':', ';', '<', '=', '>', '?', '{', '}', '|', '~':
			continue
		}

		r = append(r, s[i])
	}

	return string(r)
}
