package keydesk

import (
	"errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/google/uuid"
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
		m := models.Message{
			ID:       strfmt.UUID(v.ID.String()),
			Title:    swag.String(v.Title),
			Text:     swag.String(v.Text),
			IsRead:   v.IsRead,
			Priority: int64(v.Priority),
			Time:     strfmt.DateTime(v.CreatedAt),
		}
		if v.TTL != 0 {
			m.TTL = v.TTL.String()
		}
		ret = append(ret, &m)
	}

	return operations.NewGetMessagesOK().WithPayload(&models.Messages{
		Messages: ret,
		Total:    swag.Int64(int64(total)),
	})
}

//func CreateMessage(s message.Service, m storage.Message) middleware.Responder {
//	if err := s.CreateMessage(m.Text, m.TTL, m.Priority); err != nil {
//		return operations.NewPostUserInternalServerError()
//	}
//	return operations.NewPutMessageOK()
//}

func MarkAsRead(svc service.Service, id string) middleware.Responder {
	uid, err := uuid.Parse(id)
	if err != nil {
		return operations.NewMarkMessageAsReadDefault(http.StatusNotFound).WithPayload(&models.Error{
			Code:    http.StatusBadRequest,
			Message: swag.String(err.Error()),
		})
	}

	if err := svc.MarkAsRead(uid); err != nil {
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
