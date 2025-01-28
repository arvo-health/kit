// response_error_registry.go provides a registry to map domain-specific errors
// to ResponseErrors. This centralizes error-to-response mapping and standardizes
// error handling across the application.

package responseerror

import (
	"errors"
	"net/http"
)

// Registry maintains a mapping between errors and their corresponding ResponseErrors.
// It allows easy registration and retrieval of predefined error mappings.
type Registry struct {
	registry map[error]*ResponseError // Internal map linking errors to ResponseErrors.
}

// NewRegistry creates and initializes an empty Registry for managing error mappings.
func NewRegistry() Registry {
	return Registry{
		registry: make(map[error]*ResponseError),
	}
}

// Add associates a specific error with a ResponseError using a code and optional HTTP status.
// This method supports chaining for convenience.
func (r Registry) Add(err error, code string, status ...int) Registry {
	responseErr := New(err, code, status...)
	if responseErr == nil {
		return r // Ignore invalid ResponseError.
	}

	r.registry[err] = responseErr
	return r
}

// Get retrieves the ResponseError associated with a given error.
// If the error is not found, a default validation or internal error is returned.
func (r Registry) Get(err error) *ResponseError {
	var responseError *ResponseError

	// Check if the error is already a ResponseError.
	if errors.As(err, &responseError) {
		return responseError
	}

	// Retrieve any validation details if applicable.
	type validationsGetter interface{ Validations() map[string]string }
	var details map[string]string
	if v, ok := err.(validationsGetter); ok {
		details = v.Validations()
	}

	// Check for a direct match in the registry.
	if respError, exists := r.registry[err]; exists {
		respError.details = details
		return respError
	}

	// Handle wrapped errors (unwrapping logic).
	if respError, exists := r.registry[errors.Unwrap(err)]; exists {
		respError.message = err.Error()
		respError.details = details
		return respError
	}

	// Handle joined errors (multiple errors wrapped together).
	var joinedErrors interface{ Unwrap() []error }
	if errors.As(err, &joinedErrors) {
		unwrappedErrs := joinedErrors.Unwrap()
		for i, e := range unwrappedErrs {
			// Check if the joined error is registered.
			if respError, exists := r.registry[e]; exists {
				respError.message += ": " + unwrappedErrs[i+1].Error()
				respError.details = details
				return respError
			}
		}
	}

	// Return a default validation or internal error based on available details.
	if len(details) > 0 {
		resp := New(err, "ERR-V001", http.StatusUnprocessableEntity)
		resp.details = details
		return resp
	}

	return New(err, "ERR-I000") // Default internal server error.
}
