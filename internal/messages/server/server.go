package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	echomiddleware "github.com/oapi-codegen/echo-middleware"
	messages2 "github.com/vpngen/keydesk/gen/messages"
	jwt2 "github.com/vpngen/keydesk/internal/auth/swagger3"
	"github.com/vpngen/keydesk/internal/messages/service"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/jwt"
	"net/http"
	"time"
)

type Server struct {
	db     *storage.BrigadeStorage
	msgSvc service.Service
}

var _ messages2.StrictServerInterface = (*Server)(nil)

func SetupServer(db *storage.BrigadeStorage, authorizer jwt.Authorizer) (*echo.Echo, error) {
	swagger, err := messages2.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("get swagger: %s", err.Error())
	}

	swagger.Servers = nil

	validator := echomiddleware.OapiRequestValidatorWithOptions(
		swagger,
		&echomiddleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: jwt2.AuthFuncFactory(authorizer),
			},
		})

	e := echo.New()
	e.HideBanner = true
	e.Use(echomw.Recover(), echomw.Logger(), validator)
	messages2.RegisterHandlers(e, messages2.NewStrictHandler(NewServer(db, service.New(db)), nil))

	return e, nil
}

func NewServer(db *storage.BrigadeStorage, msgSvc service.Service) Server {
	return Server{db: db, msgSvc: msgSvc}
}

func postMessagesError(code int, message string) (messages2.PostMessagesResponseObject, error) {
	return messages2.PostMessagesdefaultJSONResponse{
		Body: messages2.Error{
			Code:    code,
			Message: message,
		},
		StatusCode: code,
	}, nil
}

func (s Server) PostMessages(_ context.Context, request messages2.PostMessagesRequestObject) (messages2.PostMessagesResponseObject, error) {
	var (
		ttl      time.Duration
		priority int
		err      error
	)
	if request.Body.Ttl != nil {
		ttl, err = time.ParseDuration(*request.Body.Ttl)
		if err != nil {
			return postMessagesError(http.StatusBadRequest, fmt.Sprintf("ttl: %s", err.Error()))
		}
	}
	if request.Body.Priority != nil {
		priority = *request.Body.Priority
	}
	msg, err := s.msgSvc.CreateMessage(request.Body.Text, ttl, priority)
	if err != nil {
		return postMessagesError(http.StatusInternalServerError, fmt.Sprintf("create message: %s", err.Error()))
	}
	return messages2.PostMessages200JSONResponse(messages2.Message{
		Id:       msg.ID,
		IsRead:   msg.IsRead,
		Priority: msg.Priority,
		Text:     msg.Text,
		Time:     msg.CreatedAt,
		Ttl:      msg.TTL.String(),
	}), nil
}

func markAsReadError(code int, message string) (messages2.PostMessagesIdReadResponseObject, error) {
	return messages2.PostMessagesIdReaddefaultJSONResponse{
		Body: messages2.Error{
			Code:    code,
			Message: message,
		},
		StatusCode: code,
	}, nil
}

func (s Server) PostMessagesIdRead(_ context.Context, request messages2.PostMessagesIdReadRequestObject) (messages2.PostMessagesIdReadResponseObject, error) {
	if err := s.msgSvc.MarkAsRead(request.Id); err != nil {
		switch {
		case errors.Is(err, service.NotFound):
			return markAsReadError(http.StatusNotFound, err.Error())
		default:
			return markAsReadError(http.StatusInternalServerError, err.Error())
		}
	}
	return messages2.PostMessagesIdRead200Response{}, nil
}

//func getSortParams(sort *Sort) (map[string]bool, error) {
//	if sort == nil {
//		return nil, nil
//	}
//	sortParams := make(map[string]bool)
//	for key, side := range *sort {
//		var desc bool
//		switch side {
//		case "asc":
//		case "desc":
//			desc = true
//		default:
//			return nil, fmt.Errorf("invalid sort direction %s", side)
//		}
//		sortParams[key] = desc
//	}
//	return sortParams, nil
//}
//
//func getMessagesError(code int, message string) (GetMessagesResponseObject, error) {
//	return GetMessagesdefaultJSONResponse{
//		Body: Error{
//			Code:    code,
//			Message: message,
//		},
//		StatusCode: code,
//	}, nil
//}
//
//func (s Server) GetMessages(ctx context.Context, request GetMessagesRequestObject) (GetMessagesResponseObject, error) {
//	sortParams, err := getSortParams(request.Params.Sort)
//	if err != nil {
//		return getMessagesError(http.StatusBadRequest, err.Error())
//	}
//
//	var priorityFilter map[string]int
//	if request.Params.Priority != nil {
//		priorityFilter = *request.Params.Priority
//	}
//
//	messages, total, err := s.msgSvc.GetMessages(
//		*request.Params.Offset,
//		*request.Params.Limit,
//		request.Params.Read,
//		priorityFilter,
//		sortParams,
//	)
//	if err != nil {
//		return getMessagesError(http.StatusInternalServerError, err.Error())
//	}
//
//	result := make([]Message, 0, len(messages))
//	for _, msg := range messages {
//		result = append(result, Message{
//			Id:       msg.ID,
//			IsRead:   msg.IsRead,
//			Priority: msg.Priority,
//			Text:     msg.Text,
//			Time:     msg.CreatedAt,
//			Ttl:      msg.TTL.String(),
//		})
//	}
//
//	return GetMessages200JSONResponse{
//		Messages: result,
//		Total:    total,
//	}, nil
//}
