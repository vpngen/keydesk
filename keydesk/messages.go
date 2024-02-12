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

func GetMessages(s message.Service, offset, limit int64, read *bool, priority *int64, priorityOp string) middleware.Responder {
	messages, total, err := s.GetMessages(offset, limit, read, priority, priorityOp)
	if err != nil {
		return operations.NewGetMessagesInternalServerError()
	}

	ret := make([]*models.Message, 0, len(messages))
	for _, v := range messages {
		ret = append(ret, &models.Message{
			Text:     swag.String(v.Text),
			IsRead:   v.IsRead,
			Priority: int64(v.Priority),
			Time:     strfmt.DateTime(v.Time),
			TTL:      v.TTL.String(),
		})
	}

	return operations.NewGetMessagesOK().WithPayload(&models.Messages{
		Messages: ret,
		Total:    int64(total),
	})
}

func CreateMessage(s message.Service, m storage.Message) middleware.Responder {
	// TODO: check if brigadier is online and send message without saving
	if err := s.CreateMessage(m.Text, m.TTL); err != nil {
		return operations.NewPostUserInternalServerError()
	}
	return operations.NewPutMessageOK()
}
