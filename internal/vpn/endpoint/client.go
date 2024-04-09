package endpoint

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

const (
	// CallTimeout - timeout for API call.
	CallTimeout = 60 * time.Second // 60 seconds.
	// ConnTimeout - timeout for API connection.
	ConnTimeout = 10 * time.Second // 10 seconds.
)

type (
	Client interface {
		PeerAdd(wgPub wgtypes.Key, params map[string]string) (APIResponse, error)
	}

	RealClient struct {
		url    url.URL
		client *http.Client
	}

	APIResponse struct {
		Code                     string `json:"code"`
		OpenvpnClientCertificate string `json:"openvpn-client-certificate"`
		Error                    string `json:"error,omitempty"`
	}
)

func NewClient(addrPort netip.AddrPort) RealClient {
	return RealClient{
		url: url.URL{
			Scheme: "http",
			Host:   addrPort.String(),
		},
		client: &http.Client{
			Transport: &http.Transport{
				//Dial:        (&net.Dialer{Timeout: ConnTimeout}).Dial,
				DialContext: (&net.Dialer{Timeout: ConnTimeout}).DialContext,
			},
		},
	}
}

func (c RealClient) addQueryParams(params map[string]string) string {
	q := c.url.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	return q.Encode()
}

var ErrInvalidRespCode = errors.New("invalid resp code")

func (c RealClient) PeerAdd(wgPub wgtypes.Key, params map[string]string) (APIResponse, error) {
	cmd := url.Values{}
	cmd.Set("peer_add", base64.StdEncoding.EncodeToString(wgPub[:]))
	c.url.RawQuery = cmd.Encode() + "&" + c.addQueryParams(params)

	log.Println("endpoint request:", c.url.String())

	res, err := c.client.Get(c.url.String())
	if err != nil {
		return APIResponse{}, fmt.Errorf("request: %w", err)
	}
	defer res.Body.Close()

	buf := bytes.NewBuffer(nil)

	_, err = io.Copy(buf, res.Body)
	if err != nil {
		return APIResponse{}, fmt.Errorf("copy body to buf: %w", err)
	}

	log.Println("endpoint response:", res.StatusCode, buf.String())

	data := APIResponse{}
	err = json.NewDecoder(buf).Decode(&data)
	if err != nil {
		return APIResponse{}, fmt.Errorf("decode: %w", err)
	}

	if data.Code != "0" {
		return APIResponse{}, fmt.Errorf("%w: %s: %s", ErrInvalidRespCode, data.Code, data.Error)
	}

	return data, nil
}
