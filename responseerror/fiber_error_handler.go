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

	// response represents the structure of the error payload sent to the client.
	type response struct {
		Code    string            `json:"code"`              // Unique error code.
		Message string            `json:"message"`           // Human-readable error message.
		Details map[string]string `json:"details,omitempty"` // Additional error details (optional).
	}

	return func(c *fiber.Ctx, err error) error {
		// Check if the error is a ResponseError, otherwise create a new generic UNKNOWN_ERROR 500 one.
		var respError *ResponseError
		if !errors.As(err, &respError) {
			respError = New(err, "UNKNOWN_ERROR")
		}

		// Send a JSON response with the appropriate HTTP status code and error details.
		return c.Status(respError.statusCode).JSON(fiber.Map{
			"error": response{
				Code:    respError.code,
				Message: respError.message,
				Details: respError.details,
			},
		})
	}
}
