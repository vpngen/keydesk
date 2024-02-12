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
	"github.com/go-openapi/swag"
)

// NewGetMessagesParams creates a new GetMessagesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetMessagesParams() *GetMessagesParams {
	return &GetMessagesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetMessagesParamsWithTimeout creates a new GetMessagesParams object
// with the ability to set a timeout on a request.
func NewGetMessagesParamsWithTimeout(timeout time.Duration) *GetMessagesParams {
	return &GetMessagesParams{
		timeout: timeout,
	}
}

// NewGetMessagesParamsWithContext creates a new GetMessagesParams object
// with the ability to set a context for a request.
func NewGetMessagesParamsWithContext(ctx context.Context) *GetMessagesParams {
	return &GetMessagesParams{
		Context: ctx,
	}
}

// NewGetMessagesParamsWithHTTPClient creates a new GetMessagesParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetMessagesParamsWithHTTPClient(client *http.Client) *GetMessagesParams {
	return &GetMessagesParams{
		HTTPClient: client,
	}
}

/*
GetMessagesParams contains all the parameters to send to the API endpoint

	for the get messages operation.

	Typically these are written to a http.Request.
*/
type GetMessagesParams struct {

	// Limit.
	//
	// Default: 25
	Limit *int64

	// Offset.
	Offset *int64

	// Priority.
	Priority *int64

	// PriorityOp.
	//
	// Default: "eq"
	PriorityOp *string

	// Read.
	Read *bool

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get messages params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetMessagesParams) WithDefaults() *GetMessagesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get messages params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetMessagesParams) SetDefaults() {
	var (
		limitDefault = int64(25)

		offsetDefault = int64(0)

		priorityOpDefault = string("eq")
	)

	val := GetMessagesParams{
		Limit:      &limitDefault,
		Offset:     &offsetDefault,
		PriorityOp: &priorityOpDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the get messages params
func (o *GetMessagesParams) WithTimeout(timeout time.Duration) *GetMessagesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get messages params
func (o *GetMessagesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get messages params
func (o *GetMessagesParams) WithContext(ctx context.Context) *GetMessagesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get messages params
func (o *GetMessagesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get messages params
func (o *GetMessagesParams) WithHTTPClient(client *http.Client) *GetMessagesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get messages params
func (o *GetMessagesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithLimit adds the limit to the get messages params
func (o *GetMessagesParams) WithLimit(limit *int64) *GetMessagesParams {
	o.SetLimit(limit)
	return o
}

// SetLimit adds the limit to the get messages params
func (o *GetMessagesParams) SetLimit(limit *int64) {
	o.Limit = limit
}

// WithOffset adds the offset to the get messages params
func (o *GetMessagesParams) WithOffset(offset *int64) *GetMessagesParams {
	o.SetOffset(offset)
	return o
}

// SetOffset adds the offset to the get messages params
func (o *GetMessagesParams) SetOffset(offset *int64) {
	o.Offset = offset
}

// WithPriority adds the priority to the get messages params
func (o *GetMessagesParams) WithPriority(priority *int64) *GetMessagesParams {
	o.SetPriority(priority)
	return o
}

// SetPriority adds the priority to the get messages params
func (o *GetMessagesParams) SetPriority(priority *int64) {
	o.Priority = priority
}

// WithPriorityOp adds the priorityOp to the get messages params
func (o *GetMessagesParams) WithPriorityOp(priorityOp *string) *GetMessagesParams {
	o.SetPriorityOp(priorityOp)
	return o
}

// SetPriorityOp adds the priorityOp to the get messages params
func (o *GetMessagesParams) SetPriorityOp(priorityOp *string) {
	o.PriorityOp = priorityOp
}

// WithRead adds the read to the get messages params
func (o *GetMessagesParams) WithRead(read *bool) *GetMessagesParams {
	o.SetRead(read)
	return o
}

// SetRead adds the read to the get messages params
func (o *GetMessagesParams) SetRead(read *bool) {
	o.Read = read
}

// WriteToRequest writes these params to a swagger request
func (o *GetMessagesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Limit != nil {

		// query param limit
		var qrLimit int64

		if o.Limit != nil {
			qrLimit = *o.Limit
		}
		qLimit := swag.FormatInt64(qrLimit)
		if qLimit != "" {

			if err := r.SetQueryParam("limit", qLimit); err != nil {
				return err
			}
		}
	}

	if o.Offset != nil {

		// query param offset
		var qrOffset int64

		if o.Offset != nil {
			qrOffset = *o.Offset
		}
		qOffset := swag.FormatInt64(qrOffset)
		if qOffset != "" {

			if err := r.SetQueryParam("offset", qOffset); err != nil {
				return err
			}
		}
	}

	if o.Priority != nil {

		// query param priority
		var qrPriority int64

		if o.Priority != nil {
			qrPriority = *o.Priority
		}
		qPriority := swag.FormatInt64(qrPriority)
		if qPriority != "" {

			if err := r.SetQueryParam("priority", qPriority); err != nil {
				return err
			}
		}
	}

	if o.PriorityOp != nil {

		// query param priority-op
		var qrPriorityOp string

		if o.PriorityOp != nil {
			qrPriorityOp = *o.PriorityOp
		}
		qPriorityOp := qrPriorityOp
		if qPriorityOp != "" {

			if err := r.SetQueryParam("priority-op", qPriorityOp); err != nil {
				return err
			}
		}
	}

	if o.Read != nil {

		// query param read
		var qrRead bool

		if o.Read != nil {
			qrRead = *o.Read
		}
		qRead := swag.FormatBool(qrRead)
		if qRead != "" {

			if err := r.SetQueryParam("read", qRead); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
