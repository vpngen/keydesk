package server

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	echomw "github.com/oapi-codegen/echo-middleware"
	messages2 "github.com/vpngen/keydesk/gen/messages"
	"github.com/vpngen/keydesk/internal/messages/service"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
)

var client *messages2.ClientWithResponses

func TestMain(m *testing.M) {
	var db storage.BrigadeStorage
	mw := func(m *testing.M) int { return m.Run() }
	mw = serverTestMiddleware(&db, mw)
	mw = storage.BrigadeTestMiddleware(&db, mw)
	mw = clientMiddleware(mw)
	os.Exit(mw(m))
}

/*
	func TestMessages(t *testing.T) {
		ctx := context.Background()

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
					res, err := client.PostMessagesWithResponse(ctx, messages2.PostMessagesJSONRequestBody{
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
	}
*/
func serverTestMiddleware(db *storage.BrigadeStorage, mw utils.TestMainMiddleware) utils.TestMainMiddleware {
	return func(m *testing.M) int {
		srv := echo.New()
		messages2.RegisterHandlers(srv, messages2.NewStrictHandler(Server{
			db:     db,
			msgSvc: service.New(db),
		}, nil))
		go func() {
			_ = srv.Start(":8000")
		}()
		ctx := context.Background()

		swagger, err := messages2.GetSwagger()
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
		c, err := messages2.NewClientWithResponses("http://localhost:8000")
		if err != nil {
			log.Fatal(err)
		}
		client = c
		return mw(m)
	}
}
