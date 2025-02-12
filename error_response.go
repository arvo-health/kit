// Package kit provides utilities for structured error handling and API response formatting.
// This file defines the ResponseError type and functions for creating and managing API-friendly error representations.

package kit

import (
	"errors"
)

// ResponseError represents a structured error used for API responses, including status, code, message, cause, and details.
type ResponseError struct {
	Code    string   `json:"code"`
	Status  int      `json:"status_code"`
	Message string   `json:"message"`
	Cause   string   `json:"cause,omitempty"`
	Details []string `json:"details,omitempty"`
	err     error
}

// NewResponseError generates a ResponseError from the provided HTTP status and error, mapping to structured error types.
func NewResponseError(status int, err error) *ResponseError {
	// check if the error is a *Error or *ValidationErrors type and map accordingly
	// if not, return a generic UNKNOWN error with the original error as the cause
	var e *Error
	if errors.As(err, &e) {
		e.Cause()
		return &ResponseError{
			Status:  status,
			Code:    e.code,
			Message: e.String(),
			Details: e.details,
			Cause:   e.Cause(),
			err:     err,
		}
	}

	var errs *ValidationErrors
	if errors.As(err, &errs) {
		return &ResponseError{
			Status:  status,
			Code:    "VALIDATION",
			Message: errs.Error(),
			Details: errs.Validations(),
			err:     err,
		}
	}

	return &ResponseError{
		Status:  status,
		Code:    "UNKNOWN",
		Message: "Ocorreu um erro inesperado. Tente novamente mais tarde ou contate o administrador.",
		Cause:   err.Error(),
		err:     err,
	}
}

// Error returns the formatted error message, optionally including the cause's message if a cause is present.
func (e *ResponseError) Error() string {
	return e.Message
}

// Unwrap returns the original error, if present.
func (e *ResponseError) Unwrap() error {
	return e.err
}
