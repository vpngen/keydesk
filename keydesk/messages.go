package keydesk

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func GetMessages(s message.Service) middleware.Responder {
	n, err := s.GetMessages()
	if err != nil {
		return operations.NewGetMessagesInternalServerError()
	}

	ret := make([]*models.Message, 0, len(n))
	for _, v := range n {
		ret = append(ret, &models.Message{
			Text:   &v.Text,
			IsRead: v.IsRead,
			Time:   strfmt.DateTime(v.Time),
		})
	}

	return operations.NewGetMessagesOK().WithPayload(ret)
}

func CreateMessage(s message.Service, m storage.Message) middleware.Responder {
	// TODO: check if brigadier is online and send message without saving
	if err := s.CreateMessage(m.Text); err != nil {
		return operations.NewPostUserInternalServerError()
	}
	return operations.NewPutMessageOK()
}
