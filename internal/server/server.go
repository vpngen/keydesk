package server

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"log"
	"net/http"
)

func NewServer(
	db *storage.BrigadeStorage,
	msgSvc message.Service,
	routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	tokenTTL int64,
) http.Handler {
	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	// create new service API
	api := operations.NewUserAPI(swaggerSpec)

	api.ServeError = errors.ServeError

	api.UseSwaggerUI()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	api.BearerAuth = keydesk.ValidateBearer(db.BrigadeID)
	api.PostTokenHandler = operations.PostTokenHandlerFunc(keydesk.CreateToken(db.BrigadeID, tokenTTL))
	api.PostUserHandler = operations.PostUserHandlerFunc(func(params operations.PostUserParams, principal interface{}) middleware.Responder {
		return keydesk.AddUser(db, params, principal, routerPublicKey, shufflerPublicKey)
	})
	api.DeleteUserUserIDHandler = operations.DeleteUserUserIDHandlerFunc(func(params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
		return keydesk.DelUserUserID(db, params, principal)
	})
	api.GetUserHandler = operations.GetUserHandlerFunc(func(params operations.GetUserParams, principal interface{}) middleware.Responder {
		return keydesk.GetUsers(db, params, principal)
	})
	api.GetUsersStatsHandler = operations.GetUsersStatsHandlerFunc(func(params operations.GetUsersStatsParams, principal interface{}) middleware.Responder {
		return keydesk.GetUsersStats(db, params, principal)
	})

	api.GetMessagesHandler = operations.GetMessagesHandlerFunc(func(params operations.GetMessagesParams) middleware.Responder {
		return keydesk.GetMessages(msgSvc)
	})
	api.PostMessageHandler = operations.PostMessageHandlerFunc(func(params operations.PostMessageParams) middleware.Responder {
		return keydesk.CreateMessage(msgSvc, storage.Message{Text: *params.Message.Text})
	})

	return api.Serve(nil)
}
