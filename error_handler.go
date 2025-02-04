// Package kit provides foundational utilities for structured error handling in Go applications.
// This file defines a custom error handler for Fiber.

package kit

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler returns a Fiber-compatible error handler that maps errors
// to structured JSON responses. If the error is not a *ResponseError,
// it wraps it in a generic UNKNOWN_ERROR with HTTP 500 status.
func ErrorHandler() fiber.ErrorHandler {

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
			respError = NewResponseError(err, "UNKNOWN_ERROR")
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
