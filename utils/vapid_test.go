package utils

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/SherClockHolmes/webpush-go"
	"io"
	"os"
	"testing"
)

func TestVapid(t *testing.T) {

	t.Log(GenerateVAPIDKeys())
}

const subscription = `{
  "endpoint": "https://updates.push.services.mozilla.com/wpush/v2/gAAAAABlosf6H7_zB85ihH3mIOKvuf_lhVQsn_oJcQqYysKycrUbkJyCfy8O5VQ5vLa9ua60L3qD_NuNVj0pKjBVcsQ9OT8kFZvwFlQd9bHMBzK8xAzduAE_FJNJawSNCsk6W-bvvQ0yDGt6UrdEjvCkIHGsa_wlGV2eCuwEaS3dOX1H5VN6xdI",
  "expirationTime": null,
  "keys": {
    "auth": "QPKJMJNGCHqki7PdnTqXvQ",
    "p256dh": "BBofkP8zv9imwXNdm3jloKqrCcWLcyMy0fvZk8OhjzV9rrIeVFmgiVJpcOULXlQzFWtwVnG0m5kTaZPHGQgaWCU"
  }
}`

func TestPush(t *testing.T) {
	s := &webpush.Subscription{}
	if err := json.Unmarshal([]byte(subscription), s); err != nil {
		t.Fatal(err)
	}

	resp, err := webpush.SendNotification([]byte("Test!"), s, &webpush.Options{
		Subscriber:      "example@example.com",
		VAPIDPublicKey:  "BI8uqN-GskHtmeqH10szMwNNR29opGc31t8d2QGRPXCwLhoEo9vY6DNYx9X147TKVQEHrAXA3BfKfVuDBE06TbE",
		VAPIDPrivateKey: "Lcw1hBkJBH2oSGevZBAp86kr4PDlQ1QxOFH8LkBNs_c",
		TTL:             60,
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp.StatusCode, resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(body))
}

func TestKeys(t *testing.T) {
	private, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	res, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(res))

	if err := pem.Encode(os.Stdout, &pem.Block{
		Type:    "PRIVATE KEY",
		Headers: nil,
		Bytes:   res,
	}); err != nil {
		t.Fatal(err)
	}

	if err := pem.Encode(os.Stdout, &pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   private.PublicKey().Bytes(),
	}); err != nil {
		t.Fatal(err)
	}
}

// Lcw1hBkJBH2oSGevZBAp86kr4PDlQ1QxOFH8LkBNs_c
//BI8uqN-GskHtmeqH10szMwNNR29opGc31t8d2QGRPXCwLhoEo9vY6DNYx9X147TKVQEHrAXA3BfKfVuDBE06TbE

//[Debug] PushSubscription (subscription.js, line 11)
//
//endpoint: "https://web.push.apple.com/QMmrSC4dbt1aUsOydli7secUxaS9TvDVouGyX80TsYtrHjEkEKmAEHdHeyegfVn26gpPzbXHVjT-N6dovpU8j0C84WNnJDiGPD3z8zenyO6xtn8X2…"
//
//expirationTime: null
//
//options: PushSubscriptionOptions
//
//applicationServerKey: ArrayBuffer {byteLength: 65, resizable: false, maxByteLength: 65}
//
//userVisibleOnly: true
//
//PushSubscriptionOptions Prototype
//
//PushSubscription Prototype
