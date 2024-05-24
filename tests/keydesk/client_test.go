package keydesk

import (
	"flag"
	clientruntime "github.com/go-openapi/runtime/client"
	"github.com/vpngen/keydesk/gen/client"
	"github.com/vpngen/keydesk/gen/client/operations"
	"os"
	"testing"
)

var host string

func TestMain(m *testing.M) {
	flag.StringVar(&host, "host", "localhost:8000", "server address")
	flag.Parse()
	os.Exit(m.Run())
}

func TestClient(t *testing.T) {
	c := client.NewHTTPClientWithConfig(nil, &client.TransportConfig{
		Host: host,
	})

	token, err := c.Operations.PostToken(operations.NewPostTokenParams())
	if err != nil {
		t.Fatal(err)
	}
	authInfo := clientruntime.BearerToken(*token.Payload.Token)

	t.Run("create user", func(t *testing.T) {
		user, err := c.Operations.PostUser(nil, authInfo)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(*user.Payload.UserName)
	})

	t.Run("get user", func(t *testing.T) {
		users, err := c.Operations.GetUser(nil, authInfo)
		if err != nil {
			t.Fatal(err)
		}
		if len(users.Payload) < 1 {
			t.Errorf("got len(users)=%d, want >=1", len(users.Payload))
		}
		for _, user := range users.Payload {
			t.Log(*user.UserID, *user.UserName)
		}
	})
}
