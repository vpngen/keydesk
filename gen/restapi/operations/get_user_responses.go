// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/vpngen/keydesk/gen/models"
)

// GetUserOKCode is the HTTP code returned for type GetUserOK
const GetUserOKCode int = 200

/*
GetUserOK A list of users.

swagger:response getUserOK
*/
type GetUserOK struct {

	/*
	  In: Body
	*/
	Payload []*models.User `json:"body,omitempty"`
}

// NewGetUserOK creates GetUserOK with default headers values
func NewGetUserOK() *GetUserOK {

	return &GetUserOK{}
}

// WithPayload adds the payload to the get user o k response
func (o *GetUserOK) WithPayload(payload []*models.User) *GetUserOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get user o k response
func (o *GetUserOK) SetPayload(payload []*models.User) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetUserOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if payload == nil {
		// return empty array
		payload = make([]*models.User, 0, 50)
	}

	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// GetUserForbiddenCode is the HTTP code returned for type GetUserForbidden
const GetUserForbiddenCode int = 403

/*
GetUserForbidden You do not have necessary permissions for the resource

swagger:response getUserForbidden
*/
type GetUserForbidden struct {
}

// NewGetUserForbidden creates GetUserForbidden with default headers values
func NewGetUserForbidden() *GetUserForbidden {

	return &GetUserForbidden{}
}

// WriteResponse to the client
func (o *GetUserForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(403)
}

// GetUserInternalServerErrorCode is the HTTP code returned for type GetUserInternalServerError
const GetUserInternalServerErrorCode int = 500

/*
GetUserInternalServerError Internal server error

swagger:response getUserInternalServerError
*/
type GetUserInternalServerError struct {
}

// NewGetUserInternalServerError creates GetUserInternalServerError with default headers values
func NewGetUserInternalServerError() *GetUserInternalServerError {

	return &GetUserInternalServerError{}
}

// WriteResponse to the client
func (o *GetUserInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(500)
}

// GetUserServiceUnavailableCode is the HTTP code returned for type GetUserServiceUnavailable
const GetUserServiceUnavailableCode int = 503

/*
GetUserServiceUnavailable Maintenance

swagger:response getUserServiceUnavailable
*/
type GetUserServiceUnavailable struct {

	/*
	  In: Body
	*/
	Payload *models.MaintenanceError `json:"body,omitempty"`
}

// NewGetUserServiceUnavailable creates GetUserServiceUnavailable with default headers values
func NewGetUserServiceUnavailable() *GetUserServiceUnavailable {

	return &GetUserServiceUnavailable{}
}

// WithPayload adds the payload to the get user service unavailable response
func (o *GetUserServiceUnavailable) WithPayload(payload *models.MaintenanceError) *GetUserServiceUnavailable {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get user service unavailable response
func (o *GetUserServiceUnavailable) SetPayload(payload *models.MaintenanceError) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetUserServiceUnavailable) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(503)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*
GetUserDefault error

swagger:response getUserDefault
*/
type GetUserDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetUserDefault creates GetUserDefault with default headers values
func NewGetUserDefault(code int) *GetUserDefault {
	if code <= 0 {
		code = 500
	}

	return &GetUserDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the get user default response
func (o *GetUserDefault) WithStatusCode(code int) *GetUserDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the get user default response
func (o *GetUserDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the get user default response
func (o *GetUserDefault) WithPayload(payload *models.Error) *GetUserDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get user default response
func (o *GetUserDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetUserDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
