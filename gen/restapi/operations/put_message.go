// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PutMessageHandlerFunc turns a function with the right signature into a put message handler
type PutMessageHandlerFunc func(PutMessageParams) middleware.Responder

// Handle executing the request and returning a response
func (fn PutMessageHandlerFunc) Handle(params PutMessageParams) middleware.Responder {
	return fn(params)
}

// PutMessageHandler interface for that can handle valid put message params
type PutMessageHandler interface {
	Handle(PutMessageParams) middleware.Responder
}

// NewPutMessage creates a new http.Handler for the put message operation
func NewPutMessage(ctx *middleware.Context, handler PutMessageHandler) *PutMessage {
	return &PutMessage{Context: ctx, Handler: handler}
}

/*
	PutMessage swagger:route PUT /messages putMessage

# Create a message

Create a message, triggered by management. If client is online, send message, else store message.
*/
type PutMessage struct {
	Context *middleware.Context
	Handler PutMessageHandler
}

func (o *PutMessage) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPutMessageParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}