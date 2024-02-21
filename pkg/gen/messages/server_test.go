package messages

import (
	"context"
	"github.com/labstack/echo/v4"
	"testing"
)

type server struct {
}

func (s server) GetMessages(ctx context.Context, request GetMessagesRequestObject) (GetMessagesResponseObject, error) {
	messages := []Message{}
	return GetMessages200JSONResponse{
		Messages: messages,
		Total:    0,
	}, nil
}

func TestServer(t *testing.T) {
	srv := echo.New()
	RegisterHandlers(srv, NewStrictHandler(server{}, nil))
	t.Log(srv.Start(":8000"))
}
