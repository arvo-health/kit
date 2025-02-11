// Package kit provides structured logging middleware for Fiber applications.
// This file defines a middleware for creating detailed and structured logs.
// It includes request, error, and response information, as well as user and context metadata.
package kit

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// LoggerMiddleware is a Fiber middleware function that logs request and response metadata using the provided logger.
// It records the request duration, attaches a unique request ID, and leverages context for logging within the request lifecycle.
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
			logError(log, c, start, err)
			return err
		}

		// Log the request and response details if no error occurred.
		logRequest(log, c, start)
		return nil
	}
}

// logRequest logs request and response metadata, including duration, user context, and grouped request/response details.
func logRequest(log *slog.Logger, c *fiber.Ctx, start time.Time) {
	end := time.Now().UTC() // Record the request end time.
	duration := end.Sub(start).Milliseconds()

	routerName := c.Route().Name
	if routerName != "" {
		routerName = routerName + ": "
	}
	// Log details grouped by request, response, and user context.
	log.Info(routerName+"request completed",
		slog.Int64("duration_ms", duration),
		getUserGroup(c),
		getRequestGroup(c, start),
		getResponseGroup(c, end))
}

// logError logs error details along with contextual information from the request, response, and user using the provided logger.
func logError(log *slog.Logger, c *fiber.Ctx, start time.Time, err error) {
	end := time.Now().UTC() // Record the request end time.
	duration := end.Sub(start).Milliseconds()

	// Check if the error is a ResponseError, otherwise create a new generic UNKNOWN error 500 one.
	var respError *ResponseError
	if !errors.As(err, &respError) {
		respError = NewResponseError(http.StatusInternalServerError, err)
	}

	routerName := c.Route().Name
	if routerName != "" {
		routerName = routerName + ": "
	}

	// Log details grouped by error, request, response, and user context.
	log.Error(routerName+respError.Error(),
		slog.Int64("duration_ms", duration),
		getErrorGroup(respError),
		getUserGroup(c),
		getRequestGroup(c, start),
		getResponseGroup(c, end, respError.Status))
}

// getUserGroup creates a user-related log group containing email, company, company category, and permissions context information.
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

// getRequestGroup groups and formats request metadata for logging, including start time, method, route, params, and queries.
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

// getResponseGroup creates a response-related log group containing end time and status, with optional override for status.
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

// getErrorGroup extracts error details from a ResponseError and structures them into a log group for consistent logging.
func getErrorGroup(respError *ResponseError) slog.Attr {
	return slog.Group("error",
		slog.String("code", respError.Code),
		slog.String("message", respError.Message),
		slog.String("cause", respError.Cause),
		slog.Any("details", respError.Details),
	)
}

// getContextValue retrieves a value of type T from the Fiber context using the specified key.
// If no value is found, it returns the provided default value.
// The function checks both the local context and the request context for the key.
func getContextValue[T any](c *fiber.Ctx, key ContextKey, defaultValue T) T {
	if value, ok := c.Locals(key).(T); ok {
		return value
	}

	if value, ok := c.Context().Value(key).(T); ok {
		return value
	}

	return defaultValue
}
