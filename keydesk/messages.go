package keydesk

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func GetMessages(s message.Service, offset, limit int64) middleware.Responder {
	n, err := s.GetMessages()
	if err != nil {
		return operations.NewGetMessagesInternalServerError()
	}

	ret := make([]*models.Message, 0, len(n))
	for _, v := range n {
		ret = append(ret, &models.Message{
			Text:   swag.String(v.Text),
			IsRead: v.IsRead,
			Time:   strfmt.DateTime(v.Time),
			TTL:    v.TTL.String(),
		})
	}

	if offset > int64(len(ret)) {
		return operations.NewGetMessagesOK().WithPayload(&models.Messages{
			Messages: nil,
			Total:    int64(len(ret)),
		})
	}

	return operations.NewGetMessagesOK().WithPayload(&models.Messages{
		Messages: ret[offset:min(offset+limit, int64(len(ret)))],
		Total:    int64(len(ret)),
	})
}

func CreateMessage(s message.Service, m storage.Message) middleware.Responder {
	// TODO: check if brigadier is online and send message without saving
	if err := s.CreateMessage(m.Text, m.TTL); err != nil {
		return operations.NewPostUserInternalServerError()
	}
	return operations.NewPutMessageOK()
}
