package keydesk

import (
	"errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/internal/messages/service"
	"net/http"
)

func GetMessages(
	s service.Service,
	offset, limit int64,
	read *bool,
	priority *int64, priorityOp string,
	sortTime, sortPriority *string,
) middleware.Responder {
	messages, total, err := s.GetMessages(offset, limit, read, priority, priorityOp, sortTime, sortPriority)
	if err != nil {
		return operations.NewGetMessagesInternalServerError()
	}

	ret := make([]*models.Message, 0, len(messages))
	for _, v := range messages {
		ret = append(ret, &models.Message{
			ID:       int64(v.ID),
			Text:     swag.String(v.Text),
			IsRead:   v.IsRead,
			Priority: int64(v.Priority),
			Time:     strfmt.DateTime(v.CreatedAt),
			TTL:      v.TTL.String(),
		})
	}

	return operations.NewGetMessagesOK().WithPayload(&models.Messages{
		Messages: ret,
		Total:    int64(total),
	})
}

//func CreateMessage(s message.Service, m storage.Message) middleware.Responder {
//	if err := s.CreateMessage(m.Text, m.TTL, m.Priority); err != nil {
//		return operations.NewPostUserInternalServerError()
//	}
//	return operations.NewPutMessageOK()
//}

func MarkAsRead(svc service.Service, id int) middleware.Responder {
	if err := svc.MarkAsRead(id); err != nil {
		switch {
		case errors.Is(err, service.NotFound):
			return operations.NewMarkMessageAsReadDefault(http.StatusNotFound).WithPayload(&models.Error{
				Code:    http.StatusNotFound,
				Message: swag.String(err.Error()),
			})
		default:
			return operations.NewMarkMessageAsReadDefault(http.StatusInternalServerError).WithPayload(&models.Error{
				Code:    http.StatusInternalServerError,
				Message: swag.String(err.Error()),
			})
		}
	}
	return operations.NewMarkMessageAsReadOK()
}
