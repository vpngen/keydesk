package messages

import (
	"context"
	"fmt"
	"github.com/go-openapi/swag"
	"github.com/labstack/echo/v4"
	echomw "github.com/oapi-codegen/echo-middleware"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var client *ClientWithResponses

func TestMain(m *testing.M) {
	var db storage.BrigadeStorage
	mw := func(m *testing.M) int { return m.Run() }
	mw = serverTestMiddleware(&db, mw)
	mw = storage.BrigadeTestMiddleware(&db, mw)
	mw = clientMiddleware(mw)
	os.Exit(mw(m))
}

func TestMessages(t *testing.T) {
	ctx := context.Background()

	t.Run("get empty messages", func(t *testing.T) {
		res, err := client.GetMessagesWithResponse(ctx, nil)
		if err != nil {
			t.Fatalf("get messages: %s", err)
		}
		if res.StatusCode() != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode())
		}
		if res.JSON200 == nil {
			t.Fatalf("expected non-nil response, got nil")
		}
		if len(res.JSON200.Messages) != 0 {
			t.Fatalf("expected 0 messages, got %d", len(res.JSON200.Messages))
		}
	})

	t.Run("create message", func(t *testing.T) {
		testCases := []struct {
			name     string
			text     string
			ttl      time.Duration
			priority int
		}{
			{"only text", "message text", 0, 0},
			{"with ttl", "message text", time.Hour, 0},
			{"with priority", "message text", 0, 10},
			{"all fields", "message text", time.Hour, 10},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				res, err := client.PostMessagesWithResponse(ctx, PostMessagesJSONRequestBody{
					Priority: &tc.priority,
					Text:     tc.text,
					Ttl:      swag.String(tc.ttl.String()),
				})
				if err != nil {
					t.Errorf("create message: %s", err)
				}
				if res.StatusCode() != http.StatusOK {
					t.Errorf("expected 200, got %d", res.StatusCode())
				}
				if res.JSON200 == nil {
					t.Errorf("expected non-nil response, got nil")
				}
				msg := *res.JSON200
				if msg.Text != tc.text {
					t.Errorf("expected text %q, got %q", tc.text, msg.Text)
				}
				if msg.Ttl != tc.ttl.String() {
					t.Errorf("expected ttl %q, got %q", tc.ttl.String(), msg.Ttl)
				}
				if msg.Priority != tc.priority {
					t.Errorf("expected priority %d, got %d", tc.priority, msg.Priority)
				}
				if msg.Time.IsZero() {
					t.Errorf("expected non-zero time, got zero")
				}
				if msg.Id == 0 {
					t.Errorf("expected non-zero id, got zero")
				}
				if msg.IsRead {
					t.Errorf("expected unread message, got read")
				}
			})
		}
	})

	t.Run("get messages", func(t *testing.T) {
		t.Run("populate", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				_, err := client.PostMessagesWithResponse(ctx, PostMessagesJSONRequestBody{
					Text:     fmt.Sprint(i + 1),
					Ttl:      swag.String((time.Duration(i+1) * time.Hour).String()),
					Priority: swag.Int((i + 1) * 100),
				})
				if err != nil {
					t.Errorf("create message: %s", err)
				}
			}
		})

		pf := PriorityFilter(map[string]int{"ge": 100})
		t.Run("paging", func(t *testing.T) {
			for i := 0; i < 5; i++ {
				res, err := client.GetMessagesWithResponse(ctx, &GetMessagesParams{
					Offset:   swag.Int(i * 10),
					Limit:    swag.Int(10),
					Priority: &pf,
				})
				if err != nil {
					t.Fatalf("get messages: %s", err)
				}
				if res.JSON200 == nil {
					t.Fatalf("expected non-nil response, got nil")
				}
				msgs := res.JSON200.Messages
				if len(msgs) != 10 {
					t.Errorf("expected 10 messages, got %d", len(msgs))
				}
				for j, msg := range msgs {
					if msg.Priority < 100 {
						t.Errorf("expected priority > 100, got %d", msg.Priority)
					}
					if fmt.Sprint(i*10+j+1) != msg.Text {
						t.Errorf("expected text %q, got %q", fmt.Sprint(i*10+j+1), msg.Text)
					}
				}
			}
		})
	})
}

func serverTestMiddleware(db *storage.BrigadeStorage, mw utils.TestMainMiddleware) utils.TestMainMiddleware {
	return func(m *testing.M) int {
		srv := echo.New()
		RegisterHandlers(srv, NewStrictHandler(Server{
			db:     db,
			msgSvc: message.New(db),
		}, nil))
		go func() {
			_ = srv.Start(":8000")
		}()
		ctx := context.Background()

		swagger, err := GetSwagger()
		if err != nil {
			log.Fatalf("Error loading swagger spec\n: %s", err)
		}
		swagger.Servers = nil
		srv.Use(echomw.OapiRequestValidator(swagger))
		code := mw(m)

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		return code
	}
}

func clientMiddleware(mw utils.TestMainMiddleware) utils.TestMainMiddleware {
	return func(m *testing.M) int {
		c, err := NewClientWithResponses("http://localhost:8000")
		if err != nil {
			log.Fatal(err)
		}
		client = c
		return mw(m)
	}
}
