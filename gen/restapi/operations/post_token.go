// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PostTokenHandlerFunc turns a function with the right signature into a post token handler
type PostTokenHandlerFunc func(PostTokenParams) middleware.Responder

// Handle executing the request and returning a response
func (fn PostTokenHandlerFunc) Handle(params PostTokenParams) middleware.Responder {
	return fn(params)
}

// PostTokenHandler interface for that can handle valid post token params
type PostTokenHandler interface {
	Handle(PostTokenParams) middleware.Responder
}

// NewPostToken creates a new http.Handler for the post token operation
func NewPostToken(ctx *middleware.Context, handler PostTokenHandler) *PostToken {
	return &PostToken{Context: ctx, Handler: handler}
}

/*
	PostToken swagger:route POST /token postToken

PostToken post token API
*/
type PostToken struct {
	Context *middleware.Context
	Handler PostTokenHandler
}

func (o *PostToken) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPostTokenParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
