package vpnapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
	"net/url"
)

type WgStatTraffic2 struct {
	Received string `json:"received"`
	Sent     string `json:"sent"`
}

type WgStatLastseen2 struct {
	Timestamp string `json:"timestamp"`
}

type WgStatEndpoint2 struct {
	Subnet string `json:"subnet"`
}

type (
	WgStatTrafficData2    map[string]WgStatTraffic2
	WgStatTrafficMap2     map[string]WgStatTrafficData2
	WgStatLastseenData2   map[string]WgStatLastseen2
	WgStatLastseenMap2    map[string]WgStatLastseenData2
	WgStatEndpointData2   map[string]WgStatEndpoint2
	WgStatEndpointMap2    map[string]WgStatEndpointData2
	WgStatAggregatedData2 map[string]int
)

type WgStatData2 struct {
	WgStatAggregatedData2 `json:"aggregated,omitempty"`
	WgStatTrafficMap2     `json:"traffic,omitempty"`
	WgStatLastseenMap2    `json:"last-seen,omitempty"`
	WgStatEndpointMap2    `json:"endpoints,omitempty"`
}

// WGStats - wg_stats endpoint-API call.
type WGStats struct {
	Code      string      `json:"code"`
	Timestamp string      `json:"timestamp"`
	Data      WgStatData2 `json:"data,omitempty"`
}

// WgPeerAdd - peer_add endpoint-API call.
func WgPeerAdd(
	actualAddrPort,
	calculatedAddrPort netip.AddrPort,
	wgPub, wgIfacePub,
	wgPSK []byte,
	ipv4,
	ipv6,
	keydesk netip.Addr,
	ovcCertRequest string,
	cloakBypasUID string,
	ipsecUsername string,
	ipsecPassword string,
	outlineSecret string,
) ([]byte, error) {
	query := fmt.Sprintf("peer_add=%s&wg-public-key=%s&wg-psk-key=%s&allowed-ips=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPSK)),
		url.QueryEscape(ipv4.String()+","+ipv6.String()),
	)

	if ovcCertRequest != "" && cloakBypasUID != "" {
		query += fmt.Sprintf("&openvpn-client-csr=%s&cloak-uid=%s",
			url.QueryEscape(ovcCertRequest),
			url.QueryEscape(cloakBypasUID),
		)
	}

	if ipsecUsername != "" && ipsecPassword != "" {
		query += fmt.Sprintf("&l2tp-username=%s&l2tp-password=%s",
			url.QueryEscape(ipsecUsername),
			url.QueryEscape(ipsecPassword),
		)
	}

	if outlineSecret != "" {
		query += fmt.Sprintf("&outline-ss-password=%s",
			url.QueryEscape(outlineSecret),
		)
	}

	if keydesk.IsValid() {
		query += fmt.Sprintf("&control-host=%s", url.QueryEscape(keydesk.String()))
	}

	body, err := getAPIRequest(actualAddrPort, calculatedAddrPort, query)
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}

	return body, nil
}

// WgPeerDel - peer_del endpoint-API call.
func WgPeerDel(actualAddrPort, calculatedAddrPort netip.AddrPort, wgPub, wgIfacePub []byte) error {
	query := fmt.Sprintf("peer_del=%s&wg-public-key=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
	)

	_, err := getAPIRequest(actualAddrPort, calculatedAddrPort, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgAdd - wg_add endpoint-API call.
func WgAdd(
	actualAddrPort,
	calculatedAddrPort netip.AddrPort,
	wgPriv []byte,
	endpointIPv4 netip.Addr,
	endpointPort uint16,
	IPv4CGNAT,
	IPv6ULA netip.Prefix,
	ovcFakeDomain string,
	ovcCACert string,
	ovcRouterCAKey string,
	ipsecPSK string,
	outlinePort uint16,
) error {
	// fmt.Fprintf(os.Stderr, "WgAdd: %d\n", len(wgPriv))

	query := fmt.Sprintf("wg_add=%s&external-ip=%s&wireguard-port=%s&internal-nets=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv)),
		url.QueryEscape(endpointIPv4.String()),
		url.QueryEscape(fmt.Sprintf("%d", endpointPort)),
		url.QueryEscape(IPv4CGNAT.String()+","+IPv6ULA.String()),
	)

	if ovcCACert != "" && len(ovcRouterCAKey) > 0 {
		query += fmt.Sprintf("&openvpn-ca-crt=%s&openvpn-ca-key=%s&cloak-domain=%s",
			url.QueryEscape(ovcCACert),
			url.QueryEscape(ovcRouterCAKey),
			url.QueryEscape(ovcFakeDomain),
		)
	}

	if ipsecPSK != "" {
		query += fmt.Sprintf("&l2tp-preshared-key=%s",
			url.QueryEscape(ipsecPSK),
		)
	}

	if outlinePort != 0 {
		query += fmt.Sprintf("&outline-ss-port=%d",
			outlinePort,
		)
	}

	_, err := getAPIRequest(actualAddrPort, calculatedAddrPort, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgDel - wg_del endpoint API call.
func WgDel(actualAddrPort, calculatedAddrPort netip.AddrPort, wgPriv []byte) error {
	query := fmt.Sprintf("wg_del=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv)),
	)

	_, err := getAPIRequest(actualAddrPort, calculatedAddrPort, query)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgStat - stat endpoint API call.
func WgStat(actualAddrPort, calculatedAddrPort netip.AddrPort, wgPub []byte) (*WGStats, error) {
	query := fmt.Sprintf("stat=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
	)

	body, err := getAPIRequest(actualAddrPort, calculatedAddrPort, query)
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}

	if body == nil {
		return nil, nil
	}

	data := &WGStats{}
	if err := json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("api payload: %w", err)
	}

	return data, nil
}
