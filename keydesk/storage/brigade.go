package storage

import (
	"fmt"
	"os"

	"github.com/vpngen/keydesk/vpnapi"
)

type BrigadeWgConfig struct {
	WgPublicKey          []byte
	WgPrivateRouterEnc   []byte
	WgPrivateShufflerEnc []byte
}

type BrigadeOvcConfig struct {
	OvcFakeDomain          string
	OvcCACertPemGzipBase64 string
	OvcRouterCAKey         string
	OvcShufflerCAKey       string
}

type BrigadeIPSecConfig struct {
	IPSecPSK            string
	IPSecPSKRouterEnc   string
	IPSecPSKShufflerEnc string
}

type BrigadeOutlineConfig struct {
	OutlinePort uint16
}

type BrigadeP0Config struct {
	P0FakeDomain string
}

// CreateBrigade - create brigade config.
func (db *BrigadeStorage) CreateBrigade(
	config *BrigadeConfig,
	wgConf *BrigadeWgConfig,
	ovcConf *BrigadeOvcConfig,
	ipcseConf *BrigadeIPSecConfig,
	outlineConf *BrigadeOutlineConfig,
	p0Conf *BrigadeP0Config,
	mode Mode,
	maxUsers uint,
) error {
	f, data, err := db.openWithoutReading(config.BrigadeID)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	db.calculatedAddrPort = vpnapi.CalcAPIAddrPort(config.EndpointIPv4)
	fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", db.calculatedAddrPort)

	switch {
	case db.APIAddrPort.Addr().IsValid() && db.APIAddrPort.Addr().IsUnspecified():
		db.actualAddrPort = db.calculatedAddrPort
	default:
		db.actualAddrPort = db.APIAddrPort
		if db.actualAddrPort.IsValid() {
			fmt.Fprintf(os.Stderr, "API endpoint: %s\n", db.actualAddrPort)
		}
	}

	data.Mode = mode
	if mode == ModeVGSocket {
		data.MaxUsers = maxUsers
	}

	data.IPv4CGNAT = config.IPv4CGNAT
	data.IPv6ULA = config.IPv6ULA
	data.DNSv4 = config.DNSIPv4
	data.DNSv6 = config.DNSIPv6
	data.EndpointIPv4 = config.EndpointIPv4
	data.EndpointDomain = config.EndpointDomain
	data.EndpointPort = config.EndpointPort
	data.KeydeskIPv6 = config.KeydeskIPv6

	data.WgPublicKey = wgConf.WgPublicKey
	data.WgPrivateRouterEnc = wgConf.WgPrivateRouterEnc
	data.WgPrivateShufflerEnc = wgConf.WgPrivateShufflerEnc

	if ovcConf != nil {
		data.CloakFakeDomain = ovcConf.OvcFakeDomain
		data.OvCAKeyRouterEnc = ovcConf.OvcRouterCAKey
		data.OvCAKeyShufflerEnc = ovcConf.OvcShufflerCAKey
		data.OvCACertPemGzipBase64 = ovcConf.OvcCACertPemGzipBase64
	}

	if ipcseConf != nil {
		data.IPSecPSK = ipcseConf.IPSecPSK
		data.IPSecPSKRouterEnc = ipcseConf.IPSecPSKRouterEnc
		data.IPSecPSKShufflerEnc = ipcseConf.IPSecPSKShufflerEnc
	}

	if outlineConf != nil {
		data.OutlinePort = outlineConf.OutlinePort
	}

	if p0Conf != nil {
		data.P0FakeDomain = p0Conf.P0FakeDomain
	}

	// if we catch a slowdown problems we need organize queue
	err = vpnapi.WgAdd(
		data.BrigadeID,
		db.actualAddrPort,
		db.calculatedAddrPort,
		data.WgPrivateRouterEnc,
		config.EndpointIPv4,
		config.EndpointPort,
		config.IPv4CGNAT,
		config.IPv6ULA,
		data.CloakFakeDomain,
		data.OvCACertPemGzipBase64,
		data.OvCAKeyRouterEnc,
		data.IPSecPSKRouterEnc,
		data.OutlinePort,
		data.P0FakeDomain,
	)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	err = commitBrigade(f, data)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// DestroyBrigade - remove brigade.
func (db *BrigadeStorage) DestroyBrigade() error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	// if we catch a slowdown problems we need organize queue
	err = vpnapi.WgDel(data.BrigadeID, db.actualAddrPort, db.calculatedAddrPort, data.WgPrivateRouterEnc)
	if err != nil {
		return fmt.Errorf("wg add: %w", err)
	}

	return nil
}

// GetVpnConfigs - get vpn configs.
func (db *BrigadeStorage) GetVpnConfigs(req *ConfigsImplemented) (*ConfigsImplemented, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if req == nil {
		req = &ConfigsImplemented{} // just for nil vectors
	}

	vpnCfgs := NewConfigsImplemented()
	vpnCfgs.NewWgConfigs(req.Wg)

	if data.OvCACertPemGzipBase64 != "" && data.OvCAKeyRouterEnc != "" && data.OvCAKeyShufflerEnc != "" {
		vpnCfgs.NewOvcConfigs(req.Ovc)
	}

	if data.IPSecPSK != "" && data.IPSecPSKRouterEnc != "" && data.IPSecPSKShufflerEnc != "" {
		vpnCfgs.NewIPSecConfigs(req.IPSec)
	}

	if data.OutlinePort > 0 {
		vpnCfgs.NewOutlineConfigs(req.Outline)
	}

	return vpnCfgs, nil
}
