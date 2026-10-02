package keydesk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVipEndpointURL(t *testing.T) {
	if got := VipEndpointURL("vip.vpn.works"); got != "https://vip.vpn.works" {
		t.Fatalf("bare host: %s", got)
	}

	if got := VipEndpointURL("http://127.0.0.1:8099/paid"); got != "http://127.0.0.1:8099/paid" {
		t.Fatalf("full url: %s", got)
	}
}

func TestFetchVipUsersPassesTokenAndStatusThrough(t *testing.T) {
	const token = "Bearer test.token.value"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}

		if r.Header.Get("Authorization") != token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"result":"error"}`))

			return
		}

		_, _ = w.Write([]byte(`{"result":"success","data":[]}`))
	}))
	defer upstream.Close()

	status, body, err := FetchVipUsers(context.Background(), upstream.Client(), upstream.URL, token)
	if err != nil || status != http.StatusOK || string(body) != `{"result":"success","data":[]}` {
		t.Fatalf("ok call: %d %s %v", status, body, err)
	}

	status, body, err = FetchVipUsers(context.Background(), upstream.Client(), upstream.URL, "Bearer wrong")
	if err != nil || status != http.StatusUnauthorized || string(body) != `{"result":"error"}` {
		t.Fatalf("rejected call: %d %s %v", status, body, err)
	}

	upstream.Close()

	if _, _, err := FetchVipUsers(context.Background(), upstream.Client(), upstream.URL, token); err == nil {
		t.Fatal("closed upstream: expected an error")
	}
}
