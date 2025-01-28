// Package middleware provides reusable middleware components for Fiber applications.
// This package includes a logging middleware for structured request and error logging.
//
// The middleware integrates with the `logger` package to log detailed information
// about incoming requests, errors, and responses.

package logger

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/arvo-health/kit"
	"github.com/arvo-health/kit/responseerror"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ResponseErrorGetter is an interface for retrieving ResponseError objects based on errors.
// This interface is used to map application errors to structured response errors.
type ResponseErrorGetter interface {
	Get(err error) *responseerror.ResponseError
}

// Middleware creates a Fiber middleware for logging incoming requests and responses.
// It logs details such as request ID, user info, error details, and response metrics.
//
// Parameters:
// - `logger`: The structured logger instance to use for logging.
// - `errorRespGetter`: An implementation of the `ResponseErrorGetter` interface.
//
// Returns:
// - A Fiber middleware handler.
func Middleware(logger *slog.Logger, errorRespGetter ResponseErrorGetter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now().UTC() // Record the request start time.

		// Generate a unique request ID and attach it to the logger context.
		requestID := uuid.NewString()
		log := logger.With(slog.String("request_id", requestID))
		c.Locals(kit.Logger, log) // Store the logger in the Fiber context for later use.

		// Call the next middleware or route handler in the chain.
		err := c.Next()
		if err != nil {
			// Retrieve and log the structured error details.
			errorResp := errorRespGetter.Get(err)
			logError(log, c, start, errorResp)
			return errorResp
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
func logError(log *slog.Logger, c *fiber.Ctx, start time.Time, errorResp *responseerror.ResponseError) {
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
	userEmail := getContextValue(c, kit.UserEmail, "unknown")
	userCompany := getContextValue(c, kit.UserCompany, "unknown")
	userCompanyCategory := getContextValue(c, kit.UserCompanyCategory, "unknown")
	userPermissions := c.Context().Value(kit.UserPermissions)

	return slog.Group("user",
		slog.String("email", userEmail),
		slog.String("company", userCompany),
		slog.String("company_category", userCompanyCategory),
		slog.String("permissions", fmt.Sprintf("%v", userPermissions)),
		// TODO: Add more user info here like role, etc.
	)
}

// getRequestGroup collects metadata about the incoming HTTP request.
// This includes the method, path, route, and query parameters.
func getRequestGroup(c *fiber.Ctx, start time.Time) slog.Attr {
	return slog.Group("request",
		slog.Time("start_time", start),
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
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

// getErrorGroup formats error details (code, category, message, etc.) for logging.
func getErrorGroup(errorResp *responseerror.ResponseError) slog.Attr {
	code, category, message, details := errorResp.DetailParts()
	return slog.Group("error",
		slog.String("code", code),
		slog.String("category", category),
		slog.String("message", message),
		slog.Any("details", details),
	)
}

// getContextValue retrieves a value from the Fiber context by its key.
// If the key is missing or the value is of a different type, it returns a default value.
func getContextValue[T any](c *fiber.Ctx, key kit.ContextKey, defaultValue T) T {
	if value, ok := c.Locals(key).(T); ok {
		return value
	}

	if value, ok := c.Context().Value(key).(T); ok {
		return value
	}

	return defaultValue
}
