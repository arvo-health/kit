package kit

import (
	"fmt"
)

// Validation represents a field-level validation error.
type Validation struct {
	Field      string
	Validation string
}

// ValidationError represents a validation-specific error with additional context.
// It includes a message, field-level validations, and an optional underlying error.
type ValidationError struct {
	message     string
	validations map[string]string
	err         error
	tmpField    string
}

// NewValidationError initializes a new validation error with the given message and optional validations.
func NewValidationError(message string, validations ...Validation) *ValidationError {
	vs := make(map[string]string, len(validations))

	if len(validations) > 0 {
		for _, v := range validations {
			vs[v.Field] = v.Validation
		}
	}

	return &ValidationError{
		message:     message,
		validations: vs,
	}
}

// Field sets the field name for the next validation error. This is useful for chaining validations.
func (e *ValidationError) Field(field string) *ValidationError {
	e.tmpField = field
	return e
}

// Err sets the validation message for the current field. This is useful for chaining validations.
func (e *ValidationError) Err(validation string) *ValidationError {
	e.validations[e.tmpField] = validation
	return e
}

// AddValidation adds a field-level validation error to the error instance.
func (e *ValidationError) AddValidation(field, validation string) *ValidationError {
	e.validations[field] = validation
	return e
}

// AddValidations adds multiple field-level validation errors to the error instance.
func (e *ValidationError) AddValidations(validations ...Validation) *ValidationError {
	for _, v := range validations {
		e.validations[v.Field] = v.Validation
	}
	return e
}

// HasValidations returns true if the error includes field-level validation errors.
func (e *ValidationError) HasValidations() bool {
	return len(e.validations) > 0
}

// Validations returns a map of field-level validation errors.
// Each entry includes the field name and its associated validation message.
func (e *ValidationError) Validations() map[string]string {
	return e.validations
}

// Error returns a string representation of the validation error message.
// If there is an underlying error, it will be included in the returned string.
func (e *ValidationError) Error() string {
	if e.err == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %s", e.message, e.err.Error())
}

// Wrap associates an underlying error with the validation error.
// This is useful for adding context or chaining errors.
func (e *ValidationError) Wrap(err error) *ValidationError {
	e.err = err
	return e
}

// Unwrap exposes the underlying error for further inspection or processing.
func (e *ValidationError) Unwrap() error {
	return e.err
}
