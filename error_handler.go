// Package kit provides foundational utilities for structured error handling in Go applications.
// This file defines a custom error handler for Fiber that transforms errors into structured JSON responses.

package kit

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// InternalErrorMessage is the only message sent to clients for errors that are not a fiber.Error or HTTPError.
const InternalErrorMessage = "internal error"

// ErrorHandler returns a Fiber-compatible error handler that maps errors to structured JSON responses.
// If the error is not a fiber.Error or HTTPError, it logs the original error and responds with a generic
// unknown-error with HTTP 500 status whose message is InternalErrorMessage, so internal failure details
// never reach the client.
func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}

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
			logUnexpectedError(logger, c, err)
			e = &HTTPError{
				Slug:    "unknown-error",
				Message: InternalErrorMessage,
				Status:  http.StatusInternalServerError,
			}
		}

		return c.Status(e.Status).JSON(Map{
			"error": e,
		})
	}
}

func logFiberError(logger *slog.Logger, c *fiber.Ctx, fiberErr *fiber.Error) {
	level := slog.LevelWarn
	if fiberErr.Code >= http.StatusInternalServerError {
		level = slog.LevelError
	}

	msg := "request failed: " + fiberErr.Message

	logger.LogAttrs(c.UserContext(), level, msg, requestLogAttr(c))
}

func logUnexpectedError(logger *slog.Logger, c *fiber.Ctx, err error) {
	logger.LogAttrs(c.UserContext(), slog.LevelError, "request failed: unexpected error",
		slog.String("error", err.Error()),
		requestLogAttr(c),
	)
}

func requestLogAttr(c *fiber.Ctx) slog.Attr {
	requestAttributes := []slog.Attr{
		slog.String("method", string(c.Context().Method())),
		slog.String("host", c.Hostname()),
		slog.String("path", c.Path()),
	}

	return slog.Attr{Key: "request", Value: slog.GroupValue(requestAttributes...)}
}
