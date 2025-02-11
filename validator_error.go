// Package kit provides tools for creating and managing structured errors in Go applications.
// This file defines the ValidationErrors struct and associated functions for managing
// validation-specific errors, including adding validations, checking their existence,
// and retrieving formatted error messages.

package kit

// ValidationErrors represents a structured validation error containing a message and details about specific violations.
type ValidationErrors struct {
	message     string
	validations []string
}

// NewValidationErrors creates a new instance of ValidationErrors with the specified message and optional validations.
func NewValidationErrors(message string, validations ...string) *ValidationErrors {
	return &ValidationErrors{
		message:     message,
		validations: validations,
	}
}

// Add appends one or more validation messages to the list of validations in the ValidationErrors instance.
func (e *ValidationErrors) Add(validation ...string) {
	e.validations = append(e.validations, validation...)
}

// Validations returns the slice of validation messages stored in the ValidationErrors instance.
func (e *ValidationErrors) Validations() []string {
	return e.validations
}

// HasValidations checks if there are any validation errors present in the ValidationErrors instance.
func (e *ValidationErrors) HasValidations() bool {
	return len(e.validations) > 0
}

// HasNoValidations returns true if there are no validation messages in the ValidationErrors instance.
func (e *ValidationErrors) HasNoValidations() bool {
	return len(e.validations) == 0
}

// ErrorOrNil returns the ValidationErrors instance if there are validation errors; otherwise, it returns nil.
func (e *ValidationErrors) ErrorOrNil() error {
	if e.HasValidations() {
		return e
	}
	return nil
}

// Error returns the error message of the ValidationErrors instance.
func (e *ValidationErrors) Error() string {
	return e.message
}
