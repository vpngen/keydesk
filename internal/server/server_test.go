package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/go-openapi/runtime"
	client2 "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/swag"
	"github.com/vpngen/keydesk/gen/client"
	"github.com/vpngen/keydesk/gen/client/operations"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/internal/auth"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/push"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
	"golang.org/x/crypto/nacl/box"
	"log"
	"net"
	"net/http"
	"os"
	"testing"
)

var kdClient client.KeydeskServer

func TestMain(m *testing.M) {
	var db storage.BrigadeStorage
	mw := func(m *testing.M) int { return m.Run() }
	mw = serverTestMiddleware(&db, mw)
	mw = storage.BrigadeTestMiddleware(&db, mw)
	mw = clientMiddleware(&kdClient, mw)
	os.Exit(mw(m))
}

func TestMessages(t *testing.T) {
	ctx := context.Background()
	var token string

	t.Run("get token", func(t *testing.T) {
		res, err := kdClient.Operations.PostToken(&operations.PostTokenParams{
			Context: ctx,
		})
		if err != nil {
			t.Fatalf("get token: %s", err)
		}
		if res.Payload.Token == nil {
			t.Fatalf("expected token, got nil")
		}
		token = *res.Payload.Token
	})

	t.Run("get empty messages", func(t *testing.T) {
		res, err := kdClient.Operations.GetMessages(&operations.GetMessagesParams{Context: ctx}, client2.BearerToken(token))
		if err != nil {
			t.Fatalf("get messages: %s", err)
		}

		if len(res.Payload) != 0 {
			t.Fatalf("expected 0 messages, got %d", len(res.Payload))
		}
	})

	t.Run("create message", func(t *testing.T) {
		buf := new(bytes.Buffer)
		text := "test"
		if err := json.NewEncoder(buf).Encode(&models.Message{
			Text: &text,
		}); err != nil {
			t.Fatalf("encode message: %s", err)
		}

		res, err := kdClient.Operations.PutMessage(&operations.PutMessageParams{
			Context: ctx,
			Message: &models.Message{Text: &text},
		})
		if err != nil {
			t.Fatalf("create message: %s", err)
		}

		if res.Code() != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, res.Code())
		}
	})

	t.Run("get messages", func(t *testing.T) {
		res, err := kdClient.Operations.GetMessages(&operations.GetMessagesParams{Context: ctx}, client2.BearerToken(token))
		if err != nil {
			t.Fatalf("get messages: %s", err)
		}

		if len(res.Payload) != 1 {
			t.Errorf("expected 1 message, got %d", len(res.Payload))
		}

		if res.Payload[0].Text == nil {
			t.Errorf("expected 'test', got nil")
		}

		if *res.Payload[0].Text != "test" {
			t.Errorf("expected 'test', got %s", *res.Payload[0].Text)
		}
	})
}

func TestPush(t *testing.T) {
	ctx := context.Background()
	t.Run("post subscription", func(t *testing.T) {
		resp, err := kdClient.Operations.PostSubscription(&operations.PostSubscriptionParams{
			Subscription: &models.Subscription{
				Endpoint: swag.String("endpoint"),
				Keys: &models.SubscriptionKeys{
					Auth:   "auth",
					P256dh: "p256dh",
				},
			},
			Context: ctx,
		})
		if err != nil {
			t.Fatalf("post subscription: %s", err)
		}

		checkSwaggerResponse(resp, t)

		if resp.Code() != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, resp.Code())
		}
	})

	t.Run("get subscription", func(t *testing.T) {
		resp, err := kdClient.Operations.GetSubscription(&operations.GetSubscriptionParams{Context: ctx})
		if err != nil {
			t.Fatalf("get subscriptions: %s", err)
		}

		checkSwaggerResponse(resp, t)

		if resp.Code() != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, resp.Code())
		}

		if swag.StringValue(resp.Payload.Endpoint) != "endpoint" {
			t.Errorf("expected 'endpoint', got %s", swag.StringValue(resp.Payload.Endpoint))
		}

		if resp.Payload.Keys.Auth != "auth" {
			t.Errorf("expected 'auth', got %s", resp.Payload.Keys.Auth)
		}

		if resp.Payload.Keys.P256dh != "p256dh" {
			t.Errorf("expected 'p256dh', got %s", resp.Payload.Keys.P256dh)
		}
	})

	t.Run("push", func(t *testing.T) {
		res, err := kdClient.Operations.SendPush(&operations.SendPushParams{
			Body: &models.PushRequest{
				Notification: &models.NotificationOptions{
					Options: &models.NotificationOptionsOptions{Body: swag.String("body")},
					Title:   swag.String("title"),
				},
				Options: &models.PushOptions{
					PrivateKey: "Lcw1hBkJBH2oSGevZBAp86kr4PDlQ1QxOFH8LkBNs_c",
					PublicKey:  "BI8uqN-GskHtmeqH10szMwNNR29opGc31t8d2QGRPXCwLhoEo9vY6DNYx9X147TKVQEHrAXA3BfKfVuDBE06TbE",
					Subscriber: "subscriber",
					Topic:      "topic",
					Urgency:    string(webpush.UrgencyHigh),
				},
				Subscription: &models.Subscription{
					Endpoint: swag.String("https://updates.push.services.mozilla.com/wpush/v2/gAAAAABlqVakh1HhXzf02cSaUUfHur0MR-he64nVH2DSC4zrILhnA_evJGahjkxIuf2cozZUzNjAczs-AH-zSdYx1r-FVll9itVAFiVm_4R5H66-ikMf1qyu03wQt7YJtTUZIOzuNxXMVZsRYXV20yu4q3FvvlJow4j3HoMad-b9lfZ6TX1NzbQ"),
					Keys: &models.SubscriptionKeys{
						Auth:   "0OfK5vsmgl5udbBY4K-Syg",
						P256dh: "BMqHXfOux6hZUnwgwjP0YHBBQvg0pGjqYj__zDTMJQJ64TP02b6HAdNZrkPn0dcEYocJkoEK7yTobnkjV-E9nwY",
					},
				},
			},
			Context: ctx,
		})
		if err != nil {
			t.Fatalf("push: %s", err)
		}

		checkSwaggerResponse(res, t)

		if res.Code() != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, res.Code())
		}
	})
}

func checkSwaggerResponse(resp runtime.ClientResponseStatus, t *testing.T) {
	if !resp.IsSuccess() {
		t.Error("expected success")
	}

	if resp.IsServerError() {
		t.Error("expected not server error")
	}

	if resp.IsClientError() {
		t.Error("expected not client error")
	}
}

func serverTestMiddleware(db *storage.BrigadeStorage, mw utils.TestMainMiddleware) utils.TestMainMiddleware {
	return func(m *testing.M) int {
		rpk, _, err := box.GenerateKey(rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		spk, _, err := box.GenerateKey(rand.Reader)
		if err != nil {
			log.Fatal(err)
		}

		api := NewServer(
			db,
			message.New(db),
			push.New(db, "", ""),
			auth.Service{
				Subject: db.BrigadeID,
				Issuer:  "test",
				Audience: []string{
					"test",
				},
			},
			rpk,
			spk,
			3600,
		)

		if err := api.Validate(); err != nil {
			log.Fatal(err)
		}

		server := &http.Server{
			Handler: api.Serve(nil),
		}

		ctx := context.Background()

		lcfg := &net.ListenConfig{}
		l, err := lcfg.Listen(ctx, "tcp4", "127.0.0.1:8000")
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			if err := server.Serve(l); err != nil {
				log.Println(err)
			}
		}()

		code := mw(m)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}

		return code
	}
}

func clientMiddleware(c *client.KeydeskServer, mw utils.TestMainMiddleware) utils.TestMainMiddleware {
	return func(m *testing.M) int {
		transport := client2.New("localhost:8000", "/", []string{"http"})
		*c = *client.Default
		c.SetTransport(transport)
		return mw(m)
	}
}