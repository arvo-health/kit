/*
Package responseerror centralizes error handling and response management.
It provides utilities to map domain-specific errors to structured HTTP responses
and to create reusable error handlers for frameworks like Fiber.

Key Features:
- Define and map domain-specific errors to HTTP responses using a registry.
- Generate structured responses with detailed metadata (code, category, message).
- Integrate with Fiber for seamless error handling and logging.
*/
package responseerror

import (
	"fmt"
	"net/http"
)

// ResponseError represents a detailed error response with metadata.
// It includes information such as code, category, HTTP status, and additional details.
type ResponseError struct {
	code       string            // Unique error code.
	category   string            // Category of the error (e.g., Validation, Business Logic).
	message    string            // Human-readable error message.
	details    map[string]string // Additional details about the error (e.g., validation issues).
	statusCode int               // HTTP status code associated with the error.
	err        error             // Underlying error, if any.
}

// New creates a new ResponseError based on an error, a unique code, and an optional HTTP status.
// If no status is provided, the default is HTTP 500 (Internal Server Error).
func New(err error, code string, status ...int) *ResponseError {
	if err == nil || code == "" {
		return nil // Ensure valid inputs.
	}

	statusCode := http.StatusInternalServerError // Default status code.
	if len(status) > 0 {
		statusCode = status[0]
	}

	return &ResponseError{
		code:       code,
		category:   deriveCategoryFromCode(code),
		message:    err.Error(),
		statusCode: statusCode,
		err:        err,
	}
}

// StatusCode sets a custom HTTP status code for the error.
// Returns the same ResponseError for method chaining.
func (e *ResponseError) StatusCode(status int) *ResponseError {
	e.statusCode = status
	return e
}

// Details adds additional metadata to the error (e.g., validation failures).
// Returns the same ResponseError for method chaining.
func (e *ResponseError) Details(details map[string]string) *ResponseError {
	e.details = details
	return e
}

// DetailParts extracts and returns the key components of the ResponseError.
// Useful for logging or structured debugging.
func (e *ResponseError) DetailParts() (code, category, message string, details []string) {
	return e.code, e.category, e.message, e.detailValues()
}

// Status retrieves the HTTP status code associated with the error.
func (e *ResponseError) Status() int {
	return e.statusCode
}

// Error implements the error interface, returning a string representation of the error.
// This is primarily used for logging and debugging.
func (e *ResponseError) Error() string {
	return fmt.Sprintf("[%s] %s", e.code, e.message)
}

// Unwrap exposes the underlying error, if any, for further inspection.
func (e *ResponseError) Unwrap() error {
	return e.err
}

// detailValues returns a slice of the detailed error information.
func (e *ResponseError) detailValues() []string {
	parts := make([]string, 0, len(e.details))
	for _, m := range e.details {
		parts = append(parts, m)
	}
	return parts
}

// deriveCategoryFromCode determines the error category based on the fifth character of the code.
// Categories include:
// - 'V': Validation
// - 'B': Business Logic
// - 'N': Not Found
// - 'R': Bad Request
// - 'P': Permission
// - 'A': Authentication
// - 'I': Internal Server
// - 'E': External Dependency
//
// If the code is invalid or unrecognized, "Unknown" is returned.
func deriveCategoryFromCode(code string) string {
	categoryMap := map[byte]string{
		'V': "validation",
		'B': "business",
		'N': "notfound",
		'R': "badrequest",
		'P': "permission",
		'A': "authentication",
		'I': "internal",
		'E': "external",
		// ...
	}

	const mincodelen = 5
	if len(code) < mincodelen {
		return "unknown" // Ensure valid code length.
	}

	// Check if the category letter exists in the map and return the category.
	if category, exists := categoryMap[code[4]]; exists {
		return category
	}

	return "unknown" // Default to unknown if the category is unrecognized.
}
