// Package kit provides utilities for structured error handling and API response formatting.
// This file defines the ResponseError type and functions for creating and managing API-friendly error representations.

package kit

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func HTTPInternalServerError(err error) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, "unknown-error", err)
}

func HTTPUnauthorizedError(err error) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, "unauthorized", err)
}

func HTTPForbiddenError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusForbidden, slug, err)
}

func HTTPBadRequestError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, slug, err)
}

func HTTPUnprocessableEntityError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, slug, err)
}

func HTTPConflictError(slug string, err error) *HTTPError {
	return NewHTTPError(http.StatusConflict, slug, err)
}

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

// NewHTTPError generates a ResponseError from the provided HTTP status and error, mapping to structured error types.
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

func (e *HTTPError) Error() string {
	return e.Message
}

func (e *HTTPError) String() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("[%s] %s (%s)", e.Slug, e.Message, strings.Join(e.Details, ","))
	}

	return fmt.Sprintf("[%s] %s", e.Slug, e.Message)
}
