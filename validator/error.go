package validator

import (
	"fmt"
)

// Validation represents a field-level validation error.
type Validation struct {
	Field      string
	Validation string
}

// Error represents a validation-specific error with additional context.
// It includes a message, field-level validations, and an optional underlying error.
type Error struct {
	message     string
	validations map[string]string
	err         error
	tmpField    string
}

// NewError initializes a new validation error with the given message and optional validations.
func NewError(message string, validations ...Validation) *Error {
	vs := make(map[string]string, len(validations))

	if len(validations) > 0 {
		for _, v := range validations {
			vs[v.Field] = v.Validation
		}
	}

	return &Error{
		message:     message,
		validations: vs,
	}
}

// Field sets the field name for the next validation error. This is useful for chaining validations.
func (e *Error) Field(field string) *Error {
	e.tmpField = field
	return e
}

// Err sets the validation message for the current field. This is useful for chaining validations.
func (e *Error) Err(validation string) *Error {
	e.validations[e.tmpField] = validation
	return e
}

// AddValidation adds a field-level validation error to the error instance.
func (e *Error) AddValidation(field, validation string) *Error {
	e.validations[field] = validation
	return e
}

// AddValidations adds multiple field-level validation errors to the error instance.
func (e *Error) AddValidations(validations ...Validation) *Error {
	for _, v := range validations {
		e.validations[v.Field] = v.Validation
	}
	return e
}

// HasValidations returns true if the error includes field-level validation errors.
func (e *Error) HasValidations() bool {
	return len(e.validations) > 0
}

// Validations returns a map of field-level validation errors.
// Each entry includes the field name and its associated validation message.
func (e *Error) Validations() map[string]string {
	return e.validations
}

// Error returns a string representation of the validation error message.
// If there is an underlying error, it will be included in the returned string.
func (e *Error) Error() string {
	if e.err == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %s", e.message, e.err.Error())
}

// Wrap associates an underlying error with the validation error.
// This is useful for adding context or chaining errors.
func (e *Error) Wrap(err error) *Error {
	e.err = err
	return e
}

// Unwrap exposes the underlying error for further inspection or processing.
func (e *Error) Unwrap() error {
	return e.err
}
