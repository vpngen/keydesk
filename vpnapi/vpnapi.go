package vpnapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
)

const endpointPort = 8080

// TemplatedAddrPort - value indicates that it is a template.
const TemplatedAddrPort = "0.0.0.0:0"

// ErrorResponse - GW response type.
type ErrorResponse struct {
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

func getAPIRequest(actualAddrPort, calculatedAddrPort netip.AddrPort, query string) ([]byte, error) {
	if !actualAddrPort.Addr().IsValid() || actualAddrPort.Addr().Compare(calculatedAddrPort.Addr()) != 0 || actualAddrPort.Port() != calculatedAddrPort.Port() {
		fmt.Fprintf(os.Stderr, "API endpoint calculated: %s\n", calculatedAddrPort)
	}

	if !actualAddrPort.Addr().IsValid() {
		fmt.Fprintf(os.Stderr, "Test Request: %s\n", &url.URL{
			Scheme:   "http",
			Host:     calculatedAddrPort.String(),
			RawQuery: query,
		})

		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "API endpoint actual: %s\n", actualAddrPort)

	apiURL := &url.URL{
		Scheme:   "http",
		Host:     actualAddrPort.String(),
		RawQuery: query,
	}

	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new req: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Request: %s\n", apiURL)

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do req: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	fmt.Fprintf(os.Stderr, "Response: %s\n", body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("resp code: %w", err)
	}

	data := &ErrorResponse{}

	err = json.Unmarshal(body, data)
	if err != nil {
		return nil, fmt.Errorf("resp body: %w", err)
	}

	if data.Code != "0" {
		return nil, fmt.Errorf("%w: %s: %s", ErrInvalidRespCode, data.Code, data.Error)
	}

	return body, nil
}
