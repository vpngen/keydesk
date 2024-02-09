// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"
)

// PutMessageOKCode is the HTTP code returned for type PutMessageOK
const PutMessageOKCode int = 200

/*
PutMessageOK OK

swagger:response putMessageOK
*/
type PutMessageOK struct {
}

// NewPutMessageOK creates PutMessageOK with default headers values
func NewPutMessageOK() *PutMessageOK {

	return &PutMessageOK{}
}

// WriteResponse to the client
func (o *PutMessageOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(200)
}

// PutMessageInternalServerErrorCode is the HTTP code returned for type PutMessageInternalServerError
const PutMessageInternalServerErrorCode int = 500

/*
PutMessageInternalServerError put message internal server error

swagger:response putMessageInternalServerError
*/
type PutMessageInternalServerError struct {
}

// NewPutMessageInternalServerError creates PutMessageInternalServerError with default headers values
func NewPutMessageInternalServerError() *PutMessageInternalServerError {

	return &PutMessageInternalServerError{}
}

// WriteResponse to the client
func (o *PutMessageInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(500)
}