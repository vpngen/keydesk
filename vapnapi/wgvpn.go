package vapnapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
	"net/url"
)

type WGStats struct {
	Code      string `json:"code"`
	Traffic   string `json:"traffic"`
	LastSeen  string `json:"last-seen"`
	Endpoints string `json:"endpoints"`
	Timestamp string `json:"timestamp"`
}

// WgPeerAdd - peer_add endpoint-API call.
func WgPeerAdd(addr netip.AddrPort, wgPub, wgIfacePub, wgPSK []byte, ipv4, ipv6, keydesk netip.Addr) error {
	query := fmt.Sprintf("peer_add=%s&wg-public-key=%s&wg-psk-key=%s&allowed-ips=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPSK)),
		url.QueryEscape(ipv4.String()+","+ipv6.String()),
	)

	if keydesk.IsValid() {
		query += fmt.Sprintf("&control-host=%s", url.QueryEscape(keydesk.String()))
	}

	_, err := getAPIRequest(addr, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgPeerDel - peer_del endpoint-API call.
func WgPeerDel(addr netip.AddrPort, wgPub, wgIfacePub []byte) error {
	query := fmt.Sprintf("peer_del=%s&wg-public-key=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
	)

	_, err := getAPIRequest(addr, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgAdd - wg_add endpoint-API call.
func WgAdd(addr netip.AddrPort, wgPriv []byte, endpointIPv4 netip.Addr, IPv4CGNAT, IPv6ULA netip.Prefix) error {
	query := fmt.Sprintf("wg_add=%s&external-ip=%s&internal-nets=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv)),
		url.QueryEscape(endpointIPv4.String()),
		url.QueryEscape(IPv4CGNAT.String()+","+IPv6ULA.String()),
	)

	_, err := getAPIRequest(addr, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgDel - wg_del endpoint API call.
func WgDel(addr netip.AddrPort, wgPriv []byte) error {
	query := fmt.Sprintf("wg_del=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv)),
	)

	_, err := getAPIRequest(addr, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgStat - stat endpoint API call.
func WgStat(addr netip.AddrPort, wgPub []byte) (*WGStats, error) {
	query := fmt.Sprintf("stat=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
	)

	body, err := getAPIRequest(addr, query)
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}

	data := &WGStats{}
	if err := json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("api payload: %w", err)
	}

	return data, nil
}
