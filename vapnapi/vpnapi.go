package vapnapi

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

// ResponsePayload - GW response type.
type ResponsePayload struct {
	Code  int    `json:"code"`
	Error string `json:"error,omitempty"`
}

var (
	// ErrInvalidRespCode - error from endpoint API.
	ErrInvalidRespCode = errors.New("invalid resp code")
)

// CalcAPIAddrPort - calc API request address and port.
func CalcAPIAddrPort(addr netip.Addr) netip.AddrPort {
	buf := [16]byte{0xfd, 0xcc, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x03}
	copy(buf[2:6], addr.AsSlice())

	return netip.AddrPortFrom(netip.AddrFrom16(buf), endpointPort)
}

func getAPIRequest(addr netip.AddrPort, query string) error {
	if !addr.Addr().IsValid() {
		fmt.Fprintf(os.Stderr, "Test Request: %s\n", &url.URL{
			Scheme:   "http",
			Host:     "localhost.local",
			RawQuery: query,
		})

		return nil
	}

	apiURL := &url.URL{
		Scheme:   "http",
		Host:     addr.String(),
		RawQuery: query,
	}

	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return fmt.Errorf("new req: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Request: %s\n", apiURL)

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("do req: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	fmt.Fprintf(os.Stderr, "Response: %s\n", body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("resp code: %w", err)
	}

	pld := &ResponsePayload{}

	err = json.Unmarshal(body, pld)
	if err != nil {
		return fmt.Errorf("resp body: %w", err)
	}

	if pld.Code != 0 {
		return fmt.Errorf("%w: %d: %s", ErrInvalidRespCode, pld.Code, pld.Error)
	}

	return nil
}
