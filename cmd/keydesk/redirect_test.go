package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectToHTTPS(t *testing.T) {
	served := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := redirectToHTTPS()(served)

	call := func(prepare func(*http.Request)) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/user?x=1", nil)
		req.Host = "vpn.works:80"
		prepare(req)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := call(func(*http.Request) {})
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "https://vpn.works/user?x=1" {
		t.Fatalf("plain http: %d %s", rec.Code, rec.Header().Get("Location"))
	}

	if rec := call(func(r *http.Request) { r.TLS = &tls.ConnectionState{} }); rec.Code != http.StatusNoContent {
		t.Fatalf("tls request redirected: %d", rec.Code)
	}

	if rec := call(func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") }); rec.Code != http.StatusNoContent {
		t.Fatalf("forwarded https redirected: %d", rec.Code)
	}

	if rec := call(func(r *http.Request) { r.Header.Set("X-Forwarded-For", "10.0.0.1") }); rec.Code != http.StatusNoContent {
		t.Fatalf("proxied request redirected: %d", rec.Code)
	}
}
