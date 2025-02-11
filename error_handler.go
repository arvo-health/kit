// Package kit provides foundational utilities for structured error handling in Go applications.
// This file defines a custom error handler for Fiber that transforms errors into structured JSON responses.

package kit

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler returns a Fiber-compatible error handler that maps errors to structured JSON responses.
// If the error is not a *ResponseError, it wraps it in a generic UNKNOWN error with HTTP 500 status.
func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {

		var respError *ResponseError
		if !errors.As(err, &respError) {
			respError = NewResponseError(http.StatusInternalServerError, err)
		}

		return c.Status(respError.Status).JSON(Map{
			"error": respError,
		})
	}
}
