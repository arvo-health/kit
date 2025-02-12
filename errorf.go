// Package kit provides tools for creating and managing structured errors in Go applications.
// This file defines the Error struct and associated functions for managing detailed errors,
// including error codes, formatted messages, error causes, and validation details.

package kit

import (
	"errors"
	"fmt"
)

// sentinel errors

// ErrBadInput signifies an error related to invalid or malformed input, typically used for validation or parsing failures.
var ErrBadInput = NewErrorf("BAD_INPUT", "Algo deu errado com os dados informados. Verifique e tente novamente.")

// ErrRequestValidation represents an error indicating that a request validation has failed.
var ErrRequestValidation = NewErrorf("REQUEST_VALIDATION", "Não foi possível concluir requisição. Verifique os dados informados e tente novamente.")

// ErrActionDenied represents an error indicating that the requested action is not permitted.
var ErrActionDenied = NewErrorf("ACTION_DENIED", "Você não tem permissão para realizar esta ação. Se precisar de acesso, contate o administrador.")

// ErrUnauthorized represents an error indicating an unauthorized access attempt.
var ErrUnauthorized = NewErrorf("UNAUTHORIZED", "Você não tem autorização para acessar este recurso.")

// Error represents a structured error with a code, message format, details, cause, and additional format arguments.
type Error struct {
	code       string
	format     string
	details    []string
	cause      error
	formatargs []any
}

// NewErrorf creates a new Error instance with a specified code, format string, and optional formatting arguments.
func NewErrorf(code string, format string, args ...any) *Error {
	return &Error{
		code:       code,
		format:     format,
		formatargs: args,
	}
}

// WithArgs appends the provided arguments to the format arguments of the Error and returns the updated Error instance.
func (e *Error) WithArgs(args ...any) *Error {
	e.formatargs = append(e.formatargs, args...)
	return e
}

// WithDetails sets the provided details to the Error and returns the updated Error instance.
func (e *Error) WithDetails(details []string) *Error {
	e.details = details
	return e
}

// WrapCause associates an external error as the cause of the Error, capturing additional validation details if applicable.
func (e *Error) WrapCause(err error) *Error {
	var validationErrs *ValidationErrors
	if errors.As(err, &validationErrs) {
		e.details = validationErrs.Validations()
	}
	e.cause = err
	return e
}

// Cause returns the error message of the associated cause, if present, or an empty string if no cause exists.
func (e *Error) Cause() string {
	if e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

// String formats the error message using the format string and arguments stored in the Error instance.
func (e *Error) String() string {
	return fmt.Sprintf(e.format, e.formatargs...)
}

// Error returns the formatted error string, including the associated cause's message if it exists.
func (e *Error) Error() string {
	if e.cause == nil {
		return e.String()
	}
	return e.String() + ": " + e.cause.Error()
}

// Unwrap returns the underlying cause of the Error, allowing access to the original error for further inspection.
func (e *Error) Unwrap() error {
	return e.cause
}

// Code returns the error code of the Error.
func (e *Error) Code() string {
	return e.code
}

// Details returns the error details of the Error.
func (e *Error) Details() []string {
	return e.details
}
