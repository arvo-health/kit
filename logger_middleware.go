// Package kit provides structured logging middleware for Fiber applications.
// This file defines a middleware that logs detailed information about incoming requests, errors, and responses.

package kit

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// LoggerMiddleware creates a Fiber middleware for logging incoming requests and responses.
// It logs details such as request ID, user info, error details, and response metrics.
//
// Parameters:
// - `logger`: The structured logger instance to use for logging.
//
// Returns:
// - A Fiber middleware handler.
func LoggerMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now().UTC() // Record the request start time.

		// Generate a unique request ID and attach it to the logger context.
		requestID := uuid.NewString()
		log := logger.With(slog.String("request_id", requestID))
		c.Locals(KeyLogger, log) // Store the logger in the Fiber context for later use.

		// Call the next middleware or route handler in the chain.
		err := c.Next()
		if err != nil {
			// Check if the error is a ResponseError, otherwise create a new generic UNKNOWN_ERROR 500 one.
			var respError *ResponseError
			if !errors.As(err, &respError) {
				respError = NewResponseError(err, "UNKNOWN_ERROR")
			}
			logError(log, c, start, respError)
			return respError
		}

		// Log the request and response details if no error occurred.
		logRequest(log, c, start)
		return nil
	}
}

// logRequest logs details about a successfully completed request.
// This includes timing metrics, user information, request metadata, and response details.
func logRequest(log *slog.Logger, c *fiber.Ctx, start time.Time) {
	end := time.Now().UTC() // Record the request end time.
	duration := end.Sub(start).Milliseconds()

	// Log details grouped by request, response, and user context.
	log.Info("request completed: "+c.Route().Name,
		slog.Int64("duration_ms", duration),
		getUserGroup(c),
		getRequestGroup(c, start),
		getResponseGroup(c, end))
}

// logError logs details about a request that resulted in an error.
// It includes error-specific details, user context, and response metadata.
func logError(log *slog.Logger, c *fiber.Ctx, start time.Time, errorResp *ResponseError) {
	end := time.Now().UTC() // Record the request end time.
	duration := end.Sub(start).Milliseconds()

	// Log details grouped by error, request, response, and user context.
	log.Error(errorResp.Error(),
		slog.Int64("duration_ms", duration),
		getErrorGroup(errorResp),
		getUserGroup(c),
		getRequestGroup(c, start),
		getResponseGroup(c, end, errorResp.Status()))
}

// getUserGroup extracts user-related metadata from the Fiber context.
// This includes information like email, company, and permissions.
func getUserGroup(c *fiber.Ctx) slog.Attr {
	userEmail := getContextValue(c, KeyUserEmail, "unknown")
	userCompany := getContextValue(c, KeyUserCompany, "unknown")
	userCompanyCategory := getContextValue(c, KeyUserCompanyCategory, "unknown")
	userPermissions := c.Context().Value(KeyUserPermissions)

	return slog.Group("user",
		slog.String("email", userEmail),
		slog.String("company", userCompany),
		slog.String("company_category", userCompanyCategory),
		slog.String("permissions", fmt.Sprintf("%v", userPermissions)),
		// TODO: Add more user info here like role, etc.
	)
}

// getRequestGroup collects metadata about the incoming HTTP request.
// This includes the method, route, and query parameters.
func getRequestGroup(c *fiber.Ctx, start time.Time) slog.Attr {
	return slog.Group("request",
		slog.Time("start_time", start),
		slog.String("method", c.Method()),
		slog.String("route", c.Route().Path),
		slog.Any("params", c.AllParams()),
		slog.Any("queries", c.Queries()),
		// TODO: Add more request info here like, request length, headers, etc.
	)
}

// getResponseGroup collects metadata about the HTTP response.
// Optionally, the HTTP status can be overridden for error responses.
func getResponseGroup(c *fiber.Ctx, end time.Time, status ...int) slog.Attr {
	statusAttr := slog.Int("status", c.Response().StatusCode())
	if len(status) > 0 {
		statusAttr = slog.Int("status", status[0])
	}

	return slog.Group("response",
		slog.Time("end_time", end),
		statusAttr,
		// TODO: Add more response info here like response length, headers, etc.
	)
}

// getErrorGroup formats error details (code, message, etc...) for logging.
func getErrorGroup(errorResp *ResponseError) slog.Attr {
	code, message, details := errorResp.DetailParts()
	return slog.Group("error",
		slog.String("code", code),
		slog.String("message", message),
		slog.Any("details", details),
	)
}

// getContextValue retrieves a value from the Fiber context by its key.
// If the key is missing or the value is of a different type, it returns a default value.
func getContextValue[T any](c *fiber.Ctx, key ContextKey, defaultValue T) T {
	if value, ok := c.Locals(key).(T); ok {
		return value
	}

	if value, ok := c.Context().Value(key).(T); ok {
		return value
	}

	return defaultValue
}
