// fiber_error_handler.go defines a custom error handler for the Fiber framework.
// It maps domain-specific errors to structured JSON responses and logs them for observability.

package responseerror

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// FiberErrorHandler creates a Fiber-compatible error handler that maps errors
// to structured JSON responses. It uses a Registry to retrieve the appropriate
// ResponseError for each error encountered.
func FiberErrorHandler(registry Registry) fiber.ErrorHandler {

	// errorResponse represents the structure of the error payload sent to the client.
	type errorResponse struct {
		Code    string            `json:"code"`              // Unique error code.
		Message string            `json:"message"`           // Human-readable error message.
		Details map[string]string `json:"details,omitempty"` // Additional error details (optional).
	}

	return func(c *fiber.Ctx, err error) error {
		var responseError *ResponseError
		if !errors.As(err, &responseError) {
			// Retrieve the ResponseError from the Registry.
			responseError = registry.Get(err)
		}

		// Send a JSON response with the appropriate HTTP status code and error details.
		return c.Status(responseError.statusCode).JSON(fiber.Map{
			"error": errorResponse{
				Code:    responseError.code,
				Message: responseError.message,
				Details: responseError.details,
			},
		})
	}
}
