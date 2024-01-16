// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// GetSubscriptionHandlerFunc turns a function with the right signature into a get subscription handler
type GetSubscriptionHandlerFunc func(GetSubscriptionParams) middleware.Responder

// Handle executing the request and returning a response
func (fn GetSubscriptionHandlerFunc) Handle(params GetSubscriptionParams) middleware.Responder {
	return fn(params)
}

// GetSubscriptionHandler interface for that can handle valid get subscription params
type GetSubscriptionHandler interface {
	Handle(GetSubscriptionParams) middleware.Responder
}

// NewGetSubscription creates a new http.Handler for the get subscription operation
func NewGetSubscription(ctx *middleware.Context, handler GetSubscriptionHandler) *GetSubscription {
	return &GetSubscription{Context: ctx, Handler: handler}
}

/*
	GetSubscription swagger:route GET /subscription getSubscription

# Get subscription

Get subscription from keydesk server
*/
type GetSubscription struct {
	Context *middleware.Context
	Handler GetSubscriptionHandler
}

func (o *GetSubscription) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetSubscriptionParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
