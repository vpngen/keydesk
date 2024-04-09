package ovc

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/amnezia"
	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/openvpn"
	"github.com/vpngen/keydesk/kdlib"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net/netip"
)

const (
	cloakProxyMethodOpenVPN = "openvpn"
	defaultCloakBrowserSig  = "chrome"
	defaultInternalDNS      = "100.126.0.1"
)

type Config struct {
	cn, bypass                                 uuid.UUID
	key                                        *ecdsa.PrivateKey
	csr                                        []byte
	routerBypass, shufflerBypass               []byte
	host, name, dns1, dns2, fakeDomain, caCert string
	ep4                                        netip.Addr
	wgPub                                      wgtypes.Key
}

func (c Config) getCloakConfig(host string, wgPub wgtypes.Key, fakeDomain string) cloak.Config {
	return cloak.NewConfig(
		host,
		wgPub.String(),
		c.bypass.String(),
		defaultCloakBrowserSig,
		cloakProxyMethodOpenVPN,
		fakeDomain,
	)
}

func (c Config) getOpenVPNConfig(ip netip.Addr, caCert, clientCert string) (openvpn.Config, error) {
	keyPem, err := c.keyPEM()
	if err != nil {
		return openvpn.Config{}, fmt.Errorf("encode key pem: %w", err)
	}

	return openvpn.Config{
		DNS:  defaultInternalDNS,
		IP:   ip.String(),
		CA:   caCert,
		Cert: clientCert,
		Key:  string(keyPem),
	}, nil
}

func (c Config) getAmneziaContainer(host string, ep4 netip.Addr, wgPub wgtypes.Key, fakeDomain, ovpnCACert, ovpnClientCert string) (amnezia.Container, error) {
	cloakCfg := c.getCloakConfig(host, wgPub, fakeDomain)
	ovpnCfg, err := c.getOpenVPNConfig(ep4, ovpnCACert, ovpnClientCert)
	if err != nil {
		return amnezia.Container{}, fmt.Errorf("get openvpn config: %w", err)
	}

	container, err := amnezia.NewOVCContainer(cloakCfg, ovpnCfg)
	if err != nil {
		return amnezia.Container{}, fmt.Errorf("amnezia container: %w", err)
	}

	return container, nil
}

func (c Config) Protocol() string {
	return vpn.IPSec
}

func (c Config) keyPKCS8() ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(c.key)
}

func (c Config) keyPEM() ([]byte, error) {
	key, err := x509.MarshalPKCS8PrivateKey(c.key)
	if err != nil {
		return nil, fmt.Errorf("key marshal: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), nil
}

func (c Config) csrPemGzBase64() ([]byte, error) {
	return kdlib.PemGzipBase64(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: c.csr})
}
