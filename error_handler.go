// Package kit provides foundational utilities for structured error handling in Go applications.
// This file defines a custom error handler for Fiber that transforms errors into structured JSON responses.

package kit

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler returns a Fiber-compatible error handler that maps errors to structured JSON responses.
// If the error is not a fiber.Error or HTTPError, it wraps it in a generic unknown-error with HTTP 500 status.
func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			logFiberError(logger, c, fiberErr)
			return c.Status(fiberErr.Code).JSON(Map{
				"error": HTTPError{
					Slug:    "fiber-err",
					Message: fiberErr.Message + ": " + fiberErr.Error(),
					Status:  fiberErr.Code,
				},
			})
		}

		var e *HTTPError
		if !errors.As(err, &e) {
			e = HTTPInternalServerError(err)
		}

		return c.Status(e.Status).JSON(Map{
			"error": e,
		})
	}
}

func logFiberError(logger *slog.Logger, c *fiber.Ctx, fiberErr *fiber.Error) {
	requestAttributes := []slog.Attr{
		slog.String("method", string(c.Context().Method())),
		slog.String("host", c.Hostname()),
		slog.String("path", c.Path()),
	}

	level := slog.LevelWarn
	if fiberErr.Code >= http.StatusInternalServerError {
		level = slog.LevelError
	}

	msg := "request failed: " + fiberErr.Message

	attributes := slog.Attr{Key: "request", Value: slog.GroupValue(requestAttributes...)}

	logger.LogAttrs(c.UserContext(), level, msg, attributes)
}
