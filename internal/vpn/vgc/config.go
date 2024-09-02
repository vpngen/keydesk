package vgc

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/ss"
	"github.com/vpngen/keydesk/internal/vpn/wg"
)

type (
	Config struct {
		Config      config     `json:"config"`
		Wireguard   wg.Config2 `json:"wireguard"`
		Cloak       cloak.VGC  `json:"cloak"`
		Shadowsocks ss.Config  `json:"shadowsocks"`
	}
	config struct {
		Version  int    `json:"version"`
		Name     string `json:"name"`
		Extended int    `json:"extended"`
	}
)

func New(name string, version, extended int, wg wg.Config2, ck cloak.VGC, ss ss.Config) Config {
	return Config{
		Config:      config{version, name, extended},
		Wireguard:   wg,
		Cloak:       ck,
		Shadowsocks: ss,
	}
}

func NewV1(name string, wg wg.Config2, ck cloak.VGC, ss ss.Config, ext int) Config {
	return New(name, 1, ext, wg, ck, ss)
}

const Schema = "vgc://"

func (c Config) Encode() (string, error) {
	buf := new(bytes.Buffer)
	gz := gzip.NewWriter(buf)
	if err := json.NewEncoder(gz).Encode(c); err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("close gzip: %w", err)
	}
	return Schema + base58.Encode(buf.Bytes()), nil
}
