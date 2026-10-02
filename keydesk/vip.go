package keydesk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk/storage"
)

const (
	VipUsersTimeout = 5 * time.Second
	vipUsersMaxBody = 1 << 20
)

var ErrVipServiceUnavailable = errors.New("VIP service unavailable")

// VipEndpointURL - the VIP service address as a URL.
func VipEndpointURL(endpoint string) string {
	if strings.Contains(endpoint, "://") {
		return endpoint
	}

	return "https://" + endpoint
}

// FetchVipUsers - POST to the VIP service, returns the upstream status and body.
func FetchVipUsers(ctx context.Context, client *http.Client, endpoint, authorization string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, VipEndpointURL(endpoint), nil)
	if err != nil {
		return 0, nil, fmt.Errorf("request: %w", err)
	}

	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %w", ErrVipServiceUnavailable, err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, vipUsersMaxBody))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: read: %w", ErrVipServiceUnavailable, err)
	}

	return resp.StatusCode, body, nil
}

// VipUsersProxy - handler of POST /vip/users.
func VipUsersProxy(db *storage.BrigadeStorage, endpoint string, client *http.Client) func(operations.PostVipUsersParams, interface{}) middleware.Responder {
	if client == nil {
		client = &http.Client{Timeout: VipUsersTimeout}
	}

	return func(params operations.PostVipUsersParams, principal interface{}) middleware.Responder {
		if !db.IsVIP() || endpoint == "" {
			return operations.NewPostVipUsersForbidden()
		}

		ctx, cancel := context.WithTimeout(params.HTTPRequest.Context(), VipUsersTimeout)
		defer cancel()

		status, body, err := FetchVipUsers(ctx, client, endpoint, params.HTTPRequest.Header.Get("Authorization"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "VIP users: %s\n", err)

			return operations.NewPostVipUsersBadGateway()
		}

		if status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "VIP users: upstream %d: %.200s\n", status, body)
		}

		return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(status)
			_, _ = w.Write(body)
		})
	}
}
