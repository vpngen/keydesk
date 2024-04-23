package vpnapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"time"
)

const endpointPort = 8080

const (
	// CallTimeout - timeout for API call.
	CallTimeout = 120 * time.Second // 60 seconds.
	// ConnTimeout - timeout for API connection.
	ConnTimeout = 5 * time.Second // 10 seconds.
)

// TemplatedAddrPort - value indicates that it is a template.
const TemplatedAddrPort = "0.0.0.0:0"

// APIResponse - GW response type.
type APIResponse struct {
	Code  string `json:"code"`
	Error string `json:"error,omitempty"`
}

// ErrInvalidRespCode - error from endpoint API.
var ErrInvalidRespCode = errors.New("invalid resp code")

// CalcAPIAddrPort - calc API request address and port.
func CalcAPIAddrPort(addr netip.Addr) netip.AddrPort {
	buf := [16]byte{0xfd, 0xcc, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x03}
	copy(buf[2:6], addr.AsSlice())

	return netip.AddrPortFrom(netip.AddrFrom16(buf), endpointPort)
}

/*
var serial uint32 = 0

func nextSerial() uint32 {
	for {
		x := atomic.AddUint32(&serial, 1)
		if x >= uint32(^uint16(0)) {
			if atomic.CompareAndSwapUint32(&serial, x, 0) {
				return 0
			}

			continue
		}

		return x
	}
}
*/

func getAPIRequest(_ string, actualAddrPort, calculatedAddrPort netip.AddrPort, query string) ([]byte, error) {
	/*
		if !actualAddrPort.Addr().IsValid() || actualAddrPort.Addr().Compare(calculatedAddrPort.Addr()) != 0 || actualAddrPort.Port() != calculatedAddrPort.Port() {
			fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", calculatedAddrPort)
		}
	*/

	if !actualAddrPort.Addr().IsValid() {
		fmt.Fprintf(os.Stderr, "Test Request: %s\n", &url.URL{
			Scheme:   "http",
			Host:     calculatedAddrPort.String(),
			RawQuery: query,
		})

		return []byte("{}"), nil
	}

	// fmt.Fprintf(os.Stderr, "API endpoint actual: %s\n", actualAddrPort)

	apiURL := &url.URL{
		Scheme:   "http",
		Host:     actualAddrPort.String(),
		RawQuery: query,
	}

	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new req: %w", err)
	}

	// num := nextSerial()
	// fmt.Fprintf(os.Stderr, "Request (%s | n=%04x): %s\n", ident, num, apiURL)

	c := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: ConnTimeout,
			}).Dial,
		},
		Timeout: CallTimeout,
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do req: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	// fmt.Fprintf(os.Stderr, "Response (%s | n=%04x): %s\n", ident, num, body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("resp code: %w", err)
	}

	data := &APIResponse{}

	err = json.Unmarshal(body, data)
	if err != nil {
		return nil, fmt.Errorf("resp body: %w", err)
	}

	if data.Code != "0" {
		return nil, fmt.Errorf("%w: %s: %s", ErrInvalidRespCode, data.Code, data.Error)
	}

	return body, nil
}
