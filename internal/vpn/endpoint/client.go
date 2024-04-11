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
	//Client interface {
	//	PeerAdd(wgPub wgtypes.Key, params map[string]string) (APIResponse, error)
	//}

	RealClient struct {
		url    url.URL
		client *http.Client
		logger *log.Logger
	}

	APIResponse struct {
		Code                     string `json:"code"`
		OpenvpnClientCertificate string `json:"openvpn-client-certificate"`
		Error                    string `json:"error,omitempty"`
	}
)

// NewClient returns endpoint client. If logger != nil, logs debug requests and responses.
func NewClient(addrPort netip.AddrPort, logger *log.Logger) RealClient {
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
		logger: logger,
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

	res, err := c.request(c.url)
	if err != nil {
		return APIResponse{}, err
	}

	defer res.Body.Close()

	data, err := c.decodeResponse(res.Body)
	if err != nil {
		return APIResponse{}, err
	}

	return data, nil
}

func (c RealClient) PeerDel(pub, epPub wgtypes.Key) error {
	q := url.Values{}
	q.Set("peer_del", pub.String())
	q.Set("wg-public-key", epPub.String())
	c.url.RawQuery = q.Encode()

	res, err := c.request(c.url)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	_, err = c.decodeResponse(res.Body)
	if err != nil {
		return err
	}

	return err
}

func (c RealClient) request(u url.URL) (*http.Response, error) {
	if c.logger != nil {
		c.logger.Println("endpoint request:", c.url.String())
	}

	res, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}

	if c.logger != nil {
		c.logger.Println("endpoint response code:", res.StatusCode)
	}

	return res, nil
}

func (c RealClient) decodeResponse(reader io.Reader) (APIResponse, error) {
	if c.logger != nil {
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, reader); err != nil {
			return APIResponse{}, fmt.Errorf("copy body to logger buf: %w", err)
		}
		c.logger.Println("endpoint response:", buf.String())
		reader = buf
	}

	data := APIResponse{}
	if err := json.NewDecoder(reader).Decode(&data); err != nil {
		return APIResponse{}, fmt.Errorf("decode: %w", err)
	}

	if data.Code != "0" {
		return APIResponse{}, fmt.Errorf("%w: %s: %s", ErrInvalidRespCode, data.Code, data.Error)
	}

	return data, nil
}
