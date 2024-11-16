package vpnapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"time"
)

const MaxDelAttempts = 3

type WgStatTrafficIn struct {
	Received string `json:"received"`
	Sent     string `json:"sent"`
}

type WgStatLastseenIn struct {
	Timestamp string `json:"timestamp"`
}

type WgStatEndpointIn struct {
	Subnet string `json:"subnet"`
}

type (
	WgStatTrafficDataIn    map[string]WgStatTrafficIn
	WgStatTrafficMapIn     map[string]WgStatTrafficDataIn
	WgStatLastseenDataIn   map[string]WgStatLastseenIn
	WgStatLastseenMapIn    map[string]WgStatLastseenDataIn
	WgStatEndpointDataIn   map[string]WgStatEndpointIn
	WgStatEndpointMapIn    map[string]WgStatEndpointDataIn
	WgStatAggregatedDataIn map[string]int
)

type WgStatDataIn struct {
	WgStatAggregatedDataIn `json:"aggregated,omitempty"`
	WgStatTrafficMapIn     `json:"traffic,omitempty"`
	WgStatLastseenMapIn    `json:"last-seen,omitempty"`
	WgStatEndpointMapIn    `json:"endpoints,omitempty"`
}

// WGStatsIn - wg_stats endpoint-API call.
type WGStatsIn struct {
	Code      string       `json:"code"`
	Timestamp string       `json:"timestamp"`
	Data      WgStatDataIn `json:"data,omitempty"`
}

// WgPeerAdd - peer_add endpoint-API call.
func WgPeerAdd(
	ident string,
	actualAddrPort,
	calculatedAddrPort netip.AddrPort,
	wgPub, wgIfacePub,
	wgPSK []byte,
	localIPv4,
	localIPv6,
	keydeskIPv6 netip.Addr,
	ovcCertRequest string,
	cloakBypasUID string,
	ipsecUsername string,
	ipsecPassword string,
	outlineSecret string,
	proto0Secret string,
) ([]byte, error) {
	query := fmt.Sprintf("peer_add=%s&wg-public-key=%s&wg-psk-key=%s&allowed-ips=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPSK)),
		url.QueryEscape(localIPv4.String()+","+localIPv6.String()),
	)

	if ovcCertRequest != "" {
		query += fmt.Sprintf("&openvpn-client-csr=%s",
			url.QueryEscape(ovcCertRequest),
		)
	}

	if cloakBypasUID != "" {
		query += fmt.Sprintf("&cloak-uid=%s",
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

	if proto0Secret != "" {
		query += fmt.Sprintf("&p0-id=%s",
			url.QueryEscape(proto0Secret),
		)
	}

	if keydeskIPv6.IsValid() {
		query += fmt.Sprintf("&control-host=%s", url.QueryEscape(keydeskIPv6.String()))
	}

	body, err := getAPIRequest(ident, actualAddrPort, calculatedAddrPort, query, CallTimeout)
	if err != nil {
		apiErr := &APIResponse{}

		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case "151":
				fmt.Fprintf(os.Stderr, "WARNING: api: %s\n", err)

				return body, nil
			}
		}

		return nil, fmt.Errorf("api: %w", err)
	}

	return body, nil
}

// WgPeerDel - peer_del endpoint-API call.
func WgPeerDel(ident string, actualAddrPort, calculatedAddrPort netip.AddrPort, wgPub, wgIfacePub []byte) error {
	query := fmt.Sprintf("peer_del=%s&wg-public-key=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPub)),
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
	)

	_, err := getAPIRequest(ident, actualAddrPort, calculatedAddrPort, query, CallTimeout)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgAdd - wg_add endpoint-API call.
func WgAdd(
	ident string,
	actualAddrPort,
	calculatedAddrPort netip.AddrPort,
	wgIfacePriv []byte,
	endpointIPv4 netip.Addr,
	endpointPort uint16,
	localNetIPv4, localNetIPv6 netip.Prefix,
	ovcFakeDomain string,
	ovcCACert string,
	ovcRouterCAKey string,
	ipsecPSK string,
	outlinePort uint16,
	proto0FakeDomain string,
) error {
	// fmt.Fprintf(os.Stderr, "WgAdd: %d\n", len(wgPriv))

	query := fmt.Sprintf("wg_add=%s&external-ip=%s&wireguard-port=%s&internal-nets=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePriv)),
		url.QueryEscape(endpointIPv4.String()),
		url.QueryEscape(fmt.Sprintf("%d", endpointPort)),
		url.QueryEscape(localNetIPv4.String()+","+localNetIPv6.String()),
	)

	if ovcFakeDomain != "" {
		query += fmt.Sprintf("&cloak-domain=%s",
			url.QueryEscape(ovcFakeDomain),
		)
	}

	if ovcCACert != "" && len(ovcRouterCAKey) > 0 {
		query += fmt.Sprintf("&openvpn-ca-crt=%s&openvpn-ca-key=%s",
			url.QueryEscape(ovcCACert),
			url.QueryEscape(ovcRouterCAKey),
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

	if proto0FakeDomain != "" {
		query += fmt.Sprintf("&p0-domain=%s",
			url.QueryEscape(proto0FakeDomain),
		)
	}

	_, err := getAPIRequest(ident, actualAddrPort, calculatedAddrPort, query, CallTimeout)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgDel - wg_del endpoint API call.
func WgDel(ident string, actualAddrPort, calculatedAddrPort netip.AddrPort, wgIfacePriv []byte) error {
	query := fmt.Sprintf("wg_del=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePriv)),
	)

	var err error

	if _, err = getAPIRequest(ident, actualAddrPort, calculatedAddrPort, query, CallTimeout); err != nil {
		apiErr := &APIResponse{}

		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case "128":
				fmt.Fprintf(os.Stderr, "WARNING: api: %s\n", err)

				return nil
			case "146":
				fmt.Fprintf(os.Stderr, "WARNING: del attempt: %s\n", err)

				<-time.After(CallTimeout)

				return nil
			}
		}

		return fmt.Errorf("api: %w", err)
	}

	return nil
}

// WgStat - stat endpoint API call.
func WgStat(ident string, actualAddrPort, calculatedAddrPort netip.AddrPort, wgIfacePub []byte) (*WGStatsIn, error) {
	query := fmt.Sprintf("stat=%s",
		url.QueryEscape(base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgIfacePub)),
	)

	body, err := getAPIRequest(ident, actualAddrPort, calculatedAddrPort, query, CallTimeout)
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}

	if body == nil {
		return nil, nil
	}

	data := &WGStatsIn{}
	if err := json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("api payload: %w", err)
	}

	return data, nil
}
