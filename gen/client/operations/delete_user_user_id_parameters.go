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
)

// NewDeleteUserUserIDParams creates a new DeleteUserUserIDParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteUserUserIDParams() *DeleteUserUserIDParams {
	return &DeleteUserUserIDParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteUserUserIDParamsWithTimeout creates a new DeleteUserUserIDParams object
// with the ability to set a timeout on a request.
func NewDeleteUserUserIDParamsWithTimeout(timeout time.Duration) *DeleteUserUserIDParams {
	return &DeleteUserUserIDParams{
		timeout: timeout,
	}
}

// NewDeleteUserUserIDParamsWithContext creates a new DeleteUserUserIDParams object
// with the ability to set a context for a request.
func NewDeleteUserUserIDParamsWithContext(ctx context.Context) *DeleteUserUserIDParams {
	return &DeleteUserUserIDParams{
		Context: ctx,
	}
}

// NewDeleteUserUserIDParamsWithHTTPClient creates a new DeleteUserUserIDParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteUserUserIDParamsWithHTTPClient(client *http.Client) *DeleteUserUserIDParams {
	return &DeleteUserUserIDParams{
		HTTPClient: client,
	}
}

/*
DeleteUserUserIDParams contains all the parameters to send to the API endpoint

	for the delete user user ID operation.

	Typically these are written to a http.Request.
*/
type DeleteUserUserIDParams struct {

	// UserID.
	UserID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete user user ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteUserUserIDParams) WithDefaults() *DeleteUserUserIDParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete user user ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteUserUserIDParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete user user ID params
func (o *DeleteUserUserIDParams) WithTimeout(timeout time.Duration) *DeleteUserUserIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete user user ID params
func (o *DeleteUserUserIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete user user ID params
func (o *DeleteUserUserIDParams) WithContext(ctx context.Context) *DeleteUserUserIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete user user ID params
func (o *DeleteUserUserIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete user user ID params
func (o *DeleteUserUserIDParams) WithHTTPClient(client *http.Client) *DeleteUserUserIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete user user ID params
func (o *DeleteUserUserIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithUserID adds the userID to the delete user user ID params
func (o *DeleteUserUserIDParams) WithUserID(userID string) *DeleteUserUserIDParams {
	o.SetUserID(userID)
	return o
}

// SetUserID adds the userId to the delete user user ID params
func (o *DeleteUserUserIDParams) SetUserID(userID string) {
	o.UserID = userID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteUserUserIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param UserID
	if err := r.SetPathParam("UserID", o.UserID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}