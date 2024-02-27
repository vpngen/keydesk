package server

import (
	"context"
	"crypto/rand"
	"fmt"
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

		if len(res.Payload.Messages) != 0 {
			t.Fatalf("expected 0 messages, got %d", len(res.Payload.Messages))
		}
	})

	//t.Run("create message", func(t *testing.T) {
	//	buf := new(bytes.Buffer)
	//	text := "test"
	//	if err := json.NewEncoder(buf).Encode(&models.Message{
	//		Text: &text,
	//	}); err != nil {
	//		t.Fatalf("encode message: %s", err)
	//	}
	//
	//	res, err := kdClient.Operations.PutMessage(&operations.PutMessageParams{
	//		Context: ctx,
	//		Message: &models.Message{Text: &text, TTL: "5m"},
	//	})
	//	if err != nil {
	//		t.Fatalf("create message: %s", err)
	//	}
	//
	//	if res.Code() != http.StatusOK {
	//		t.Errorf("expected status code %d, got %d", http.StatusOK, res.Code())
	//	}
	//})

	t.Run("get messages", func(t *testing.T) {
		res, err := kdClient.Operations.GetMessages(&operations.GetMessagesParams{
			Context: ctx,
		}, client2.BearerToken(token))
		if err != nil {
			t.Fatalf("get messages: %s", err)
		}

		if len(res.Payload.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(res.Payload.Messages))
		}

		if res.Payload.Messages[0].Text == nil {
			t.Errorf("expected 'test', got nil")
		}

		if *res.Payload.Messages[0].Text != "test" {
			t.Errorf("expected 'test', got %s", *res.Payload.Messages[0].Text)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		//t.Run("create 100 messages", func(t *testing.T) {
		//	for i := 0; i < 100; i++ {
		//		res, err := kdClient.Operations.PutMessage(&operations.PutMessageParams{
		//			Context: ctx,
		//			Message: &models.Message{
		//				Text: swag.String(fmt.Sprintf("test-%d", i)),
		//				TTL:  "5m",
		//			},
		//		})
		//		if err != nil {
		//			t.Fatalf("create message: %s", err)
		//		}
		//		if !res.IsSuccess() {
		//			t.Errorf("expected status code %d, got %d", http.StatusOK, res.Code())
		//		}
		//	}
		//})

		for _, perPage := range []int{10, 25, 50} {
			for _, page := range []int{1, 2, 5, 10} {
				t.Run(fmt.Sprintf("limit %d, offset %d", perPage, page), func(t *testing.T) {
					offset := 1
					res, err := kdClient.Operations.GetMessages(&operations.GetMessagesParams{
						Limit:   swag.Int64(int64(perPage)),
						Offset:  swag.Int64(int64((page-1)*perPage + offset)),
						Context: ctx,
					}, client2.BearerToken(token))
					if err != nil {
						t.Fatalf("get messages: %s", err)
					}

					if res.Payload.Total != 100 {
						t.Errorf("expected total %d messages, got %d", 100, res.Payload.Total)
					}

					if len(res.Payload.Messages) > perPage {
						t.Errorf("expected <= %d messages, got %d", perPage, len(res.Payload.Messages))
					}

					for i, m := range res.Payload.Messages {
						if m.Text == nil {
							t.Errorf("expected 'test', got nil")
						}
						if *m.Text != fmt.Sprintf("test-%d", (page-1)*perPage+i) {
							t.Errorf("expected 'test-%d', got %s", (page-1)*perPage+i, *m.Text)
						}
					}
				})
			}
		}
	})

	t.Run("filters", func(t *testing.T) {
		t.Run("read", func(t *testing.T) {
			tests := []struct {
				name    string
				read    bool
				wantLen int
			}{
				{"true", true, 0},
				{"false", false, 25},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					res, err := kdClient.Operations.GetMessages(
						&operations.GetMessagesParams{
							Read:    swag.Bool(test.read),
							Context: ctx,
						},
						client2.BearerToken(token),
					)
					if err != nil {
						t.Fatalf("get messages: %s", err)
					}
					if len(res.Payload.Messages) != test.wantLen {
						t.Errorf("expected total %d messages, got %d", test.wantLen, len(res.Payload.Messages))
					}
				})
			}
		})

		t.Run("priority", func(t *testing.T) {
			tests := []struct {
				name     string
				op       string
				priority int
				wantLen  int
			}{
				{"==0", "eq", 0, 25},
				{"!=0", "ne", 0, 0},
				{">0", "gt", 0, 0},
				{"<0", "lt", 0, 0},
				{">=0", "ge", 0, 25},
				{"<=0", "le", 0, 25},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					res, err := kdClient.Operations.GetMessages(
						&operations.GetMessagesParams{
							Priority:   swag.Int64(int64(test.priority)),
							PriorityOp: swag.String(test.op),
							Context:    ctx,
						},
						client2.BearerToken(token),
					)
					if err != nil {
						t.Fatalf("get messages: %s", err)
					}
					if len(res.Payload.Messages) != test.wantLen {
						t.Errorf("expected total %d messages, got %d", test.wantLen, len(res.Payload.Messages))
					}
				})
			}
		})
	})

	t.Run("sorting", func(t *testing.T) {
		//t.Run("create messages", func(t *testing.T) {
		//	now := time.Now()
		//	for i := 0; i < 25; i++ {
		//		_, err := kdClient.Operations.PutMessage(&operations.PutMessageParams{
		//			Message: &models.Message{
		//				Text:     swag.String(fmt.Sprintf("test-%d", i)),
		//				Time:     strfmt.DateTime(now.Add(time.Duration(i) * time.Minute)),
		//				Priority: int64(i),
		//				TTL:      fmt.Sprintf("%dm", i),
		//			},
		//			Context: ctx,
		//		})
		//		if err != nil {
		//			t.Fatalf("post message: %s", err)
		//		}
		//	}
		//})

		t.Run("get messages", func(t *testing.T) {
			tests := []struct {
				name         string
				sortTime     *string
				sortPriority *string
			}{
				{"no sort", nil, nil},
				{"time asc", swag.String("asc"), nil},
				{"time desc", swag.String("desc"), nil},
				{"priority asc", nil, swag.String("asc")},
				{"priority desc", nil, swag.String("desc")},
				{"time, priority asc", swag.String("asc"), swag.String("asc")},
				{"time asc, priority desc", swag.String("asc"), swag.String("desc")},
				{"time, priority desc", swag.String("desc"), swag.String("desc")},
				{"time desc, priority asc", swag.String("desc"), swag.String("asc")},
			}
			if token == "" {
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
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					res, err := kdClient.Operations.GetMessages(
						&operations.GetMessagesParams{
							SortTime:     test.sortTime,
							SortPriority: test.sortPriority,
							Context:      ctx,
						},
						client2.BearerToken(token),
					)
					if err != nil {
						t.Fatalf("get messages: %s", err)
					}
					if len(res.Payload.Messages) != 25 {
						t.Errorf("expected total %d messages, got %d", 25, len(res.Payload.Messages))
					}
					t.Log()
					t.Log(test.name)
					for _, m := range res.Payload.Messages {
						t.Log(*m.Text, m.Time, m.Priority, m.TTL)
					}
				})
			}
		})
	})

	t.Run("mark as read", func(t *testing.T) {
		if token == "" {
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
		}

		var id int

		t.Run("get last message", func(t *testing.T) {
			res, err := kdClient.Operations.GetMessages(
				&operations.GetMessagesParams{
					Context: ctx,
					Offset:  swag.Int64(0),
					Limit:   swag.Int64(1),
					Read:    swag.Bool(false),
				},
				client2.BearerToken(token),
			)
			if err != nil {
				t.Fatalf("get messages: %s", err)
			}
			if len(res.Payload.Messages) != 1 {
				t.Fatalf("expected total %d messages, got %d", 1, len(res.Payload.Messages))
			}
			if res.Payload.Messages[0].IsRead {
				t.Errorf("expected unread message, got read")
			}
			if res.Payload.Messages[0].ID == 0 {
				t.Errorf("expected message id, got 0")
			}
			id = int(res.Payload.Messages[0].ID)
		})

		t.Run("mark", func(t *testing.T) {
			res, err := kdClient.Operations.MarkMessageAsRead(
				&operations.MarkMessageAsReadParams{
					Context: ctx,
					ID:      int64(id),
				},
				client2.BearerToken(token),
			)
			if err != nil {
				t.Fatalf("mark message as read: %s", err)
			}
			if res.Code() != http.StatusOK {
				t.Errorf("expected status code %d, got %d", http.StatusOK, res.Code())
			}
		})

		t.Run("check message is read", func(t *testing.T) {
			res, err := kdClient.Operations.GetMessages(
				&operations.GetMessagesParams{
					Context: ctx,
					Offset:  swag.Int64(0),
					Limit:   swag.Int64(1),
					Read:    swag.Bool(true),
				},
				client2.BearerToken(token),
			)
			if err != nil {
				t.Fatalf("get messages: %s", err)
			}
			if len(res.Payload.Messages) != 1 {
				t.Fatalf("expected total %d messages, got %d", 1, len(res.Payload.Messages))
			}
			if !res.Payload.Messages[0].IsRead {
				t.Errorf("expected read message is read, got unread")
			}
		})
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
