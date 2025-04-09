// Package kit provides foundational utilities for structured error handling in Go applications.
// This file defines a custom error handler for Fiber that transforms errors into structured JSON responses.

package kit

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler returns a Fiber-compatible error handler that maps errors to structured JSON responses.
// If the error is not a fiber.Error or HTTPError, it wraps it in a generic unknown-error with HTTP 500 status.
func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {

		//var fiberErr *fiber.Error
		//if errors.As(err, &fiberErr) {
		//	return c.Status(fiberErr.Code).JSON(Map{
		//		"error": HTTPError{
		//			Slug:    "fiber-err",
		//			Message: fiberErr.Message + fiberErr.Error(),
		//			Status:  fiberErr.Code,
		//		},
		//	})
		//}

		var e *HTTPError
		if !errors.As(err, &e) {
			e = HTTPInternalServerError(err)
		}

		return c.Status(e.Status).JSON(Map{
			"error": e,
		})
	}
}
