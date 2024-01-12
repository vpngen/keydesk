package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
	"golang.org/x/crypto/nacl/box"
	"log"
	"net"
	"net/http"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	var db storage.BrigadeStorage
	mw := func(m *testing.M) int { return m.Run() }
	mw = serverTestMiddleware(&db, mw)
	mw = utils.BrigadeTestMiddleware(&db, mw)
	os.Exit(mw(m))
}

func TestMessages(t *testing.T) {
	t.Run("get empty messages", func(t *testing.T) {
		resp, err := http.Get("http://127.0.0.1:8000/messages")
		if err != nil {
			t.Errorf("get messages: %s", err)
		}
		defer resp.Body.Close()

		var messages models.Messages

		if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
			t.Errorf("decode messages: %s", err)
		}

		if len(messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(messages))
		}
	})

	t.Run("create message", func(t *testing.T) {
		buf := new(bytes.Buffer)
		text := "test"
		if err := json.NewEncoder(buf).Encode(&models.Message{
			Text: &text,
		}); err != nil {
			t.Errorf("encode message: %s", err)
		}

		resp, err := http.Post("http://127.0.0.1:8000/messages", "application/json", buf)
		if err != nil {
			t.Errorf("create message: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("get messages", func(t *testing.T) {
		resp, err := http.Get("http://127.0.0.1:8000/messages")
		if err != nil {
			t.Errorf("get messages: %s", err)
		}
		defer resp.Body.Close()

		var messages models.Messages

		if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
			t.Errorf("decode messages: %s", err)
		}

		if len(messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(messages))
		}

		if messages[0].Text == nil {
			t.Errorf("expected 'test', got nil")
		}

		if *messages[0].Text != "test" {
			t.Errorf("expected 'test', got %s", *messages[0].Text)
		}
	})
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

		server := &http.Server{
			Handler: NewServer(
				db,
				message.New(db),
				rpk,
				spk,
				3600,
			),
		}

		ctx := context.Background()
		//ctx, cancel := context.WithCancel(context.Background())
		//defer cancel()

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
