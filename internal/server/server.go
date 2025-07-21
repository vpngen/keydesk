package server

import (
	"log"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	goSwagger "github.com/vpngen/keydesk/internal/auth/go-swagger"
	"github.com/vpngen/keydesk/internal/messages/service"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/vpngine/naclkey"
)

func NewServer(
	db *storage.BrigadeStorage,
	msgSvc service.Service,
	issuer jwt.KeydeskTokenIssuer,
	goSwaggerAuth goSwagger.Service,
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

	api.PostTokenHandler = operations.PostTokenHandlerFunc(keydesk.CreateToken(db, issuer, tokenTTL))

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

	api.PatchUserUserIDBlockHandler = operations.PatchUserUserIDBlockHandlerFunc(func(params operations.PatchUserUserIDBlockParams, principal interface{}) middleware.Responder {
		return keydesk.BlockUserUserID(db, params, principal)
	})

	api.PatchUserUserIDUnblockHandler = operations.PatchUserUserIDUnblockHandlerFunc(func(params operations.PatchUserUserIDUnblockParams, principal interface{}) middleware.Responder {
		return keydesk.UnblockUserUserID(db, params, principal)
	})

	api.GetMessagesHandler = operations.GetMessagesHandlerFunc(func(params operations.GetMessagesParams, principal interface{}) middleware.Responder {
		return keydesk.GetMessages(
			msgSvc,
			*params.Offset,
			*params.Limit,
			params.Read,
			params.Priority,
			*params.PriorityOp,
			params.SortTime,
			params.SortPriority,
		)
	})
	api.MarkMessageAsReadHandler = operations.MarkMessageAsReadHandlerFunc(func(params operations.MarkMessageAsReadParams, principal interface{}) middleware.Responder {
		return keydesk.MarkAsRead(msgSvc, params.ID.String())
	})

	//api.PostSubscriptionHandler = operations.PostSubscriptionHandlerFunc(func(params operations.PostSubscriptionParams) middleware.Responder {
	//	return keydesk.PostSubscription(pushSvc, webpush.Subscription{
	//		Endpoint: swag.StringValue(params.Subscription.Endpoint),
	//		Keys: webpush.Keys{
	//			P256dh: params.Subscription.Keys.P256dh,
	//			Auth:   params.Subscription.Keys.Auth,
	//		},
	//	})
	//})
	//api.GetSubscriptionHandler = operations.GetSubscriptionHandlerFunc(func(params operations.GetSubscriptionParams) middleware.Responder {
	//	return keydesk.GetSubscription(pushSvc)
	//})
	//api.SendPushHandler = operations.SendPushHandlerFunc(pushSvc.SendPushHandler)

	// api.APIKeyAuthenticator = goSwaggerAuth.APIKeyAuthenticator

	api.BearerAuth = goSwaggerAuth.BearerAuth

	// api.APIAuthorizer = runtime.AuthorizerFunc(goSwaggerAuth.Authorize)

	return api
}
