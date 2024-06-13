package vgc

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
)

type (
	Config struct {
		Config      config      `json:"config"`
		Wireguard   Wireguard   `json:"wireguard"`
		Cloak       Cloak       `json:"cloak"`
		Shadowsocks Shadowsocks `json:"shadowsocks"`
	}
	config struct {
		Version  int    `json:"version"`
		Name     string `json:"name"`
		Extended int    `json:"extended"`
	}
)

func New(name string, version, extended int, wg Wireguard, ck Cloak, ss Shadowsocks) Config {
	return Config{
		Config:      config{version, name, extended},
		Wireguard:   wg,
		Cloak:       ck,
		Shadowsocks: ss,
	}
}

func NewV1(name string, wg Wireguard, ck Cloak, ss Shadowsocks) Config {
	return New(name, 1, 1, wg, ck, ss)
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
