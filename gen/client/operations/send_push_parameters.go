// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/vpngen/keydesk/gen/models"
)

// NewSendPushParams creates a new SendPushParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewSendPushParams() *SendPushParams {
	return &SendPushParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewSendPushParamsWithTimeout creates a new SendPushParams object
// with the ability to set a timeout on a request.
func NewSendPushParamsWithTimeout(timeout time.Duration) *SendPushParams {
	return &SendPushParams{
		timeout: timeout,
	}
}

// NewSendPushParamsWithContext creates a new SendPushParams object
// with the ability to set a context for a request.
func NewSendPushParamsWithContext(ctx context.Context) *SendPushParams {
	return &SendPushParams{
		Context: ctx,
	}
}

// NewSendPushParamsWithHTTPClient creates a new SendPushParams object
// with the ability to set a custom HTTPClient for a request.
func NewSendPushParamsWithHTTPClient(client *http.Client) *SendPushParams {
	return &SendPushParams{
		HTTPClient: client,
	}
}

/*
SendPushParams contains all the parameters to send to the API endpoint

	for the send push operation.

	Typically these are written to a http.Request.
*/
type SendPushParams struct {

	// Body.
	Body *models.PushRequest

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the send push params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *SendPushParams) WithDefaults() *SendPushParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the send push params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *SendPushParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the send push params
func (o *SendPushParams) WithTimeout(timeout time.Duration) *SendPushParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the send push params
func (o *SendPushParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the send push params
func (o *SendPushParams) WithContext(ctx context.Context) *SendPushParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the send push params
func (o *SendPushParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the send push params
func (o *SendPushParams) WithHTTPClient(client *http.Client) *SendPushParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the send push params
func (o *SendPushParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the send push params
func (o *SendPushParams) WithBody(body *models.PushRequest) *SendPushParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the send push params
func (o *SendPushParams) SetBody(body *models.PushRequest) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *SendPushParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}