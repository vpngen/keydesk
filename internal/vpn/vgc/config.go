package vgc

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/proto0"
	"github.com/vpngen/keydesk/internal/vpn/ss"
	"github.com/vpngen/keydesk/internal/vpn/wg"
)

type (
	Config struct {
		Config      config         `json:"config"`
		Wireguard   *wg.Config2    `json:"wireguard,omitempty"`
		Cloak       *cloak.VGC     `json:"cloak,omitempty"`
		Shadowsocks *ss.Config     `json:"shadowsocks,omitempty"`
		Proto0      *proto0.Config `json:"protocol0,omitempty"`
	}

	config struct {
		Version  int    `json:"version"`
		Name     string `json:"name"`
		Domain   string `json:"domain"`
		Extended int    `json:"extended"`
	}
)

func New(name, domain string, version, extended int, wg *wg.Config2, ck *cloak.VGC, ss *ss.Config, proto0 *proto0.Config) Config {
	return Config{
		Config: config{
			Version:  version,
			Name:     name,
			Domain:   domain,
			Extended: extended,
		},
		Wireguard:   wg,
		Cloak:       ck,
		Shadowsocks: ss,
		Proto0:      proto0,
	}
}

func NewV1(name, domain string, wg *wg.Config2, ck *cloak.VGC, ss *ss.Config, proto0 *proto0.Config, ext int) Config {
	return New(name, domain, 1, ext, wg, ck, ss, proto0)
}

func (c Config) Encode() (string, error) {
	buf := new(bytes.Buffer)
	gz := gzip.NewWriter(buf)
	if err := json.NewEncoder(gz).Encode(c); err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("close gzip: %w", err)
	}
	return base58.Encode(buf.Bytes()), nil
}
