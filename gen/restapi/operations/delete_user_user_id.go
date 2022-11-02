// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// DeleteUserUserIDHandlerFunc turns a function with the right signature into a delete user user ID handler
type DeleteUserUserIDHandlerFunc func(DeleteUserUserIDParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn DeleteUserUserIDHandlerFunc) Handle(params DeleteUserUserIDParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// DeleteUserUserIDHandler interface for that can handle valid delete user user ID params
type DeleteUserUserIDHandler interface {
	Handle(DeleteUserUserIDParams, interface{}) middleware.Responder
}

// NewDeleteUserUserID creates a new http.Handler for the delete user user ID operation
func NewDeleteUserUserID(ctx *middleware.Context, handler DeleteUserUserIDHandler) *DeleteUserUserID {
	return &DeleteUserUserID{Context: ctx, Handler: handler}
}

/*
	DeleteUserUserID swagger:route DELETE /user/{UserID} deleteUserUserId

DeleteUserUserID delete user user ID API
*/
type DeleteUserUserID struct {
	Context *middleware.Context
	Handler DeleteUserUserIDHandler
}

func (o *DeleteUserUserID) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewDeleteUserUserIDParams()
	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		*r = *aCtx
	}
	var principal interface{}
	if uprinc != nil {
		principal = uprinc.(interface{}) // this is really a interface{}, I promise
	}

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params, principal) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
