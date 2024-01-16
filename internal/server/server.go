package server

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/internal/auth"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/push"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"log"
)

func NewServer(
	db *storage.BrigadeStorage,
	msgSvc message.Service,
	pushSvc push.Service,
	authSvc auth.Service,
	routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	tokenTTL int64,
) *operations.UserAPI {
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

	api.PostTokenHandler = operations.PostTokenHandlerFunc(keydesk.CreateToken(db.BrigadeID, tokenTTL, []string{"messages:get"})) // TODO: get scopes from request

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

	api.GetMessagesHandler = operations.GetMessagesHandlerFunc(func(params operations.GetMessagesParams, principal interface{}) middleware.Responder {
		return keydesk.GetMessages(msgSvc)
	})
	api.PutMessageHandler = operations.PutMessageHandlerFunc(func(params operations.PutMessageParams) middleware.Responder {
		return keydesk.CreateMessage(msgSvc, storage.Message{Text: *params.Message.Text})
	})

	api.PostSubscriptionHandler = operations.PostSubscriptionHandlerFunc(func(params operations.PostSubscriptionParams) middleware.Responder {
		return keydesk.PostSubscription(pushSvc, storage.PushSubscription{
			Endpoint: params.Subscription.Endpoint,
			Keys: storage.Keys{
				P256DH: params.Subscription.Keys.P256dh,
				Auth:   params.Subscription.Keys.Auth,
			},
		})
	})

	api.GetSubscriptionHandler = operations.GetSubscriptionHandlerFunc(func(params operations.GetSubscriptionParams) middleware.Responder {
		return keydesk.GetSubscription(pushSvc)
	})

	api.APIKeyAuthenticator = authSvc.APIKeyAuthenticator
	api.BearerAuth = authSvc.BearerAuth
	api.APIAuthorizer = runtime.AuthorizerFunc(authSvc.Authorize)

	return api
}
