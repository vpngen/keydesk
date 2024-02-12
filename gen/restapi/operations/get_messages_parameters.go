// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// NewGetMessagesParams creates a new GetMessagesParams object
// with the default values initialized.
func NewGetMessagesParams() GetMessagesParams {

	var (
		// initialize parameters with default values

		limitDefault  = int64(25)
		offsetDefault = int64(0)

		priorityOpDefault = string("eq")
	)

	return GetMessagesParams{
		Limit: &limitDefault,

		Offset: &offsetDefault,

		PriorityOp: &priorityOpDefault,
	}
}

// GetMessagesParams contains all the bound params for the get messages operation
// typically these are obtained from a http.Request
//
// swagger:parameters getMessages
type GetMessagesParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  In: query
	  Default: 25
	*/
	Limit *int64
	/*
	  In: query
	  Default: 0
	*/
	Offset *int64
	/*
	  In: query
	*/
	Priority *int64
	/*
	  In: query
	  Default: "eq"
	*/
	PriorityOp *string
	/*
	  In: query
	*/
	Read *bool
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetMessagesParams() beforehand.
func (o *GetMessagesParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qLimit, qhkLimit, _ := qs.GetOK("limit")
	if err := o.bindLimit(qLimit, qhkLimit, route.Formats); err != nil {
		res = append(res, err)
	}

	qOffset, qhkOffset, _ := qs.GetOK("offset")
	if err := o.bindOffset(qOffset, qhkOffset, route.Formats); err != nil {
		res = append(res, err)
	}

	qPriority, qhkPriority, _ := qs.GetOK("priority")
	if err := o.bindPriority(qPriority, qhkPriority, route.Formats); err != nil {
		res = append(res, err)
	}

	qPriorityOp, qhkPriorityOp, _ := qs.GetOK("priority-op")
	if err := o.bindPriorityOp(qPriorityOp, qhkPriorityOp, route.Formats); err != nil {
		res = append(res, err)
	}

	qRead, qhkRead, _ := qs.GetOK("read")
	if err := o.bindRead(qRead, qhkRead, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindLimit binds and validates parameter Limit from query.
func (o *GetMessagesParams) bindLimit(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		// Default values have been previously initialized by NewGetMessagesParams()
		return nil
	}

	value, err := swag.ConvertInt64(raw)
	if err != nil {
		return errors.InvalidType("limit", "query", "int64", raw)
	}
	o.Limit = &value

	return nil
}

// bindOffset binds and validates parameter Offset from query.
func (o *GetMessagesParams) bindOffset(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		// Default values have been previously initialized by NewGetMessagesParams()
		return nil
	}

	value, err := swag.ConvertInt64(raw)
	if err != nil {
		return errors.InvalidType("offset", "query", "int64", raw)
	}
	o.Offset = &value

	return nil
}

// bindPriority binds and validates parameter Priority from query.
func (o *GetMessagesParams) bindPriority(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		return nil
	}

	value, err := swag.ConvertInt64(raw)
	if err != nil {
		return errors.InvalidType("priority", "query", "int64", raw)
	}
	o.Priority = &value

	return nil
}

// bindPriorityOp binds and validates parameter PriorityOp from query.
func (o *GetMessagesParams) bindPriorityOp(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		// Default values have been previously initialized by NewGetMessagesParams()
		return nil
	}
	o.PriorityOp = &raw

	if err := o.validatePriorityOp(formats); err != nil {
		return err
	}

	return nil
}

// validatePriorityOp carries on validations for parameter PriorityOp
func (o *GetMessagesParams) validatePriorityOp(formats strfmt.Registry) error {

	if err := validate.EnumCase("priority-op", "query", *o.PriorityOp, []interface{}{"eq", "ne", "gt", "lt", "ge", "le"}, true); err != nil {
		return err
	}

	return nil
}

// bindRead binds and validates parameter Read from query.
func (o *GetMessagesParams) bindRead(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		return nil
	}

	value, err := swag.ConvertBool(raw)
	if err != nil {
		return errors.InvalidType("read", "query", "bool", raw)
	}
	o.Read = &value

	return nil
}
