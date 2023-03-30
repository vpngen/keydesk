// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// User user
//
// swagger:model user
type User struct {

	// last visit a s country
	LastVisitASCountry string `json:"LastVisitASCountry,omitempty"`

	// last visit a s name
	LastVisitASName string `json:"LastVisitASName,omitempty"`

	// last visit hour
	// Format: date-time
	LastVisitHour *strfmt.DateTime `json:"LastVisitHour,omitempty"`

	// last visit subnet
	LastVisitSubnet string `json:"LastVisitSubnet,omitempty"`

	// monthly quota remaining g b
	MonthlyQuotaRemainingGB float32 `json:"MonthlyQuotaRemainingGB,omitempty"`

	// person desc
	PersonDesc string `json:"PersonDesc,omitempty"`

	// person desc link
	PersonDescLink string `json:"PersonDescLink,omitempty"`

	// person name
	PersonName string `json:"PersonName,omitempty"`

	// problems
	Problems []string `json:"Problems"`

	// status
	Status string `json:"Status,omitempty"`

	// throttling till
	// Format: date-time
	ThrottlingTill *strfmt.DateTime `json:"ThrottlingTill,omitempty"`

	// user ID
	// Required: true
	UserID *string `json:"UserID"`

	// user name
	// Required: true
	UserName *string `json:"UserName"`
}

// Validate validates this user
func (m *User) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLastVisitHour(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateThrottlingTill(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUserID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUserName(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *User) validateLastVisitHour(formats strfmt.Registry) error {
	if swag.IsZero(m.LastVisitHour) { // not required
		return nil
	}

	if err := validate.FormatOf("LastVisitHour", "body", "date-time", m.LastVisitHour.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *User) validateThrottlingTill(formats strfmt.Registry) error {
	if swag.IsZero(m.ThrottlingTill) { // not required
		return nil
	}

	if err := validate.FormatOf("ThrottlingTill", "body", "date-time", m.ThrottlingTill.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *User) validateUserID(formats strfmt.Registry) error {

	if err := validate.Required("UserID", "body", m.UserID); err != nil {
		return err
	}

	return nil
}

func (m *User) validateUserName(formats strfmt.Registry) error {

	if err := validate.Required("UserName", "body", m.UserName); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this user based on context it is used
func (m *User) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *User) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *User) UnmarshalBinary(b []byte) error {
	var res User
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
