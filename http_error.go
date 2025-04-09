// Package kit provides utilities for structured error handling and API response formatting.
// This file defines the HTTPError type and functions for creating and managing API-friendly error representations.

package kit

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// HTTPInternalServerError creates an HTTPError with status 500 and slug "unknown-error" for the given error.
func HTTPInternalServerError(err error) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, "unknown-error", err)
}

// HTTPUnauthorizedError creates an HTTPError with status 401 and slug "unauthorized" for the given error.
func HTTPUnauthorizedError(err error) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, "unauthorized", err)
}

// HTTPForbiddenError creates an HTTPError with status code 403 (Forbidden), a slug, and an optional underlying error.
func HTTPForbiddenError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusForbidden, slug, err)
}

// HTTPBadRequestError returns a new HTTPError with a 400 status code, representing a bad request error.
func HTTPBadRequestError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, slug, err)
}

// HTTPUnprocessableEntityError returns an HTTPError with status 422, a custom slug, and an optional error message/details.
func HTTPUnprocessableEntityError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, slug, err)
}

// HTTPConflictError creates an HTTPError with a 409 Conflict status, slug, and optional error for additional context.
func HTTPConflictError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusConflict, slug, err)
}

// HTTPNotFoundError creates an HTTPError with a 404 status, a provided slug, and an error message.
func HTTPNotFoundError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusNotFound, slug, err)
}

// HTTPError represents a structured error used for API responses, including status, code, message, cause, and details.
type HTTPError struct {
	Slug    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	Status  int      `json:"status_code"`
}

// NewHTTPError generates a HTTPError from the provided HTTP status and error, mapping to structured error types.
func NewHTTPError(status int, slug string, err error) *HTTPError {
	if status == 0 {
		status = 500
	}

	if slug == "" {
		slug = "unknown-error"
	}

	if err == nil {
		err = errors.New("unknown error")
	}

	var details []string
	var validationErrs *ValidationErrors
	if errors.As(err, &validationErrs) {
		details = validationErrs.Validations()
	}

	return &HTTPError{
		Status:  status,
		Slug:    slug,
		Message: err.Error(),
		Details: details,
	}
}

// Error returns the error message contained within the HTTPError structure.
func (e *HTTPError) Error() string {
	return e.Message
}

// String returns a formatted string representation of the HTTPError, including slug, message, and optional details.
func (e *HTTPError) String() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("[%s] %s (%s)", e.Slug, e.Message, strings.Join(e.Details, ","))
	}

	return fmt.Sprintf("[%s] %s", e.Slug, e.Message)
}
