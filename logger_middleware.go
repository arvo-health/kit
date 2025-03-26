// Package kit provides structured logging middleware for Fiber applications.
// This file defines a middleware for creating detailed and structured logs.
// It includes request, error, and response information, as well as user and context metadata.
package kit

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type customAttributesCtxKeyType struct{}

var (
	customAttributesCtxKey = customAttributesCtxKeyType{}

	HiddenRequestHeaders = map[string]struct{}{
		"authorization": {},
		"cookie":        {},
		"set-cookie":    {},
		"x-auth-token":  {},
		"x-csrf-token":  {},
		"x-xsrf-token":  {},
	}
	HiddenResponseHeaders = map[string]struct{}{
		"set-cookie": {},
	}

	RequestIDHeaderKey = "X-Request-Id"
)

type Config struct {
	DefaultLevel       slog.Level
	WithUserAgent      bool
	WithRequestHeader  bool
	WithResponseHeader bool
}

func LoggerMiddleware(logger *slog.Logger) fiber.Handler {
	return LoggerMiddlewareWithConfig(logger, Config{
		DefaultLevel:       slog.LevelInfo,
		WithUserAgent:      false,
		WithRequestHeader:  false,
		WithResponseHeader: false,
	})
}

// LoggerMiddlewareWithConfig is a Fiber middleware function that logs request and response metadata using the provided logger.
// It records the request duration, attaches a unique request ID, and leverages context for logging within the request lifecycle.
func LoggerMiddlewareWithConfig(logger *slog.Logger, config Config) fiber.Handler {
	var (
		once       sync.Once
		errHandler fiber.ErrorHandler
	)

	return func(c *fiber.Ctx) error {
		once.Do(func() {
			errHandler = c.App().ErrorHandler
		})

		start := time.Now().UTC()
		path := c.Path()
		query := string(c.Request().URI().QueryString())

		requestID := c.Get(RequestIDHeaderKey)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Context().SetUserValue(CtxKeyRequestID, requestID)

		c.Set("X-Request-ID", requestID)

		log := logger.With(slog.String("request_id", requestID))
		c.Locals(CtxKeyLogger, log) // Store the logger in the Fiber context for later use.

		var errmsg string

		err := c.Next()
		if err != nil {
			errmsg = err.Error()
			if err = errHandler(c, err); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError) //nolint:errcheck
			}
		}

		status := c.Response().StatusCode()
		method := c.Context().Method()
		host := c.Hostname()
		params := c.AllParams()
		route := c.Route().Path
		end := time.Now().UTC()
		latency := end.Sub(start)
		userAgent := c.Context().UserAgent()

		baseAttributes := []slog.Attr{
			slog.String("request_id", requestID),
		}

		requestAttributes := []slog.Attr{
			slog.Time("time", start.UTC()),
			slog.String("method", string(method)),
			slog.String("host", host),
			slog.String("path", path),
			slog.String("query", query),
			slog.Any("params", params),
			slog.String("route", route),
			slog.Int("length", len(c.Body())),
		}

		responseAttributes := []slog.Attr{
			slog.Time("time", end.UTC()),
			slog.Duration("latency", latency),
			slog.Int("status", status),
			slog.Int("length", len(c.Response().Body())),
		}

		userAttributes := []slog.Attr{
			slog.String("email", getContextValue(c, CtxKeyUserEmail, "unknown")),
			slog.String("company", getContextValue(c, CtxKeyUserCompany, "unknown")),
			slog.String("company_category", getContextValue(c, CtxKeyUserCompanyCategory, "unknown")),
			slog.String("permissions", fmt.Sprintf("%v", c.Context().Value(CtxKeyUserPermissions))),
			// TODO: add user role
		}

		// request headers
		if config.WithRequestHeader {
			kv := []any{}

			for k, v := range c.GetReqHeaders() {
				if _, found := HiddenRequestHeaders[strings.ToLower(k)]; found {
					continue
				}
				kv = append(kv, slog.Any(k, v))
			}

			requestAttributes = append(requestAttributes, slog.Group("header", kv...))
		}

		if config.WithUserAgent {
			requestAttributes = append(requestAttributes, slog.String("user-agent", string(userAgent)))
		}

		// response headers
		if config.WithResponseHeader {
			kv := []any{}

			for k, v := range c.GetRespHeaders() {
				if _, found := HiddenResponseHeaders[strings.ToLower(k)]; found {
					continue
				}
				kv = append(kv, slog.Any(k, v))
			}

			responseAttributes = append(responseAttributes, slog.Group("header", kv...))
		}

		msg := c.Route().Name
		if msg != "" {
			msg += ": "
		}

		level := config.DefaultLevel
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
			msg += "request failed: " + errmsg
		} else if status >= http.StatusBadRequest {
			level = slog.LevelWarn
			msg += "request failed: " + errmsg
		} else {
			msg += "request succeeded"
		}

		attributes := append(baseAttributes,
			slog.Attr{Key: "request", Value: slog.GroupValue(requestAttributes...)},
			slog.Attr{Key: "response", Value: slog.GroupValue(responseAttributes...)},
			slog.Attr{Key: "user", Value: slog.GroupValue(userAttributes...)},
		)

		// custom context values
		if v := c.Context().UserValue(customAttributesCtxKey); v != nil {
			switch attrs := v.(type) {
			case []slog.Attr:
				attributes = append(attributes, attrs...)
			}
		}

		logger.LogAttrs(c.UserContext(), level, msg, attributes...)

		return err
	}
}

// AddCustomAttributes adds slog custom attributes to the request context.
func AddCustomAttributes(c *fiber.Ctx, attr slog.Attr) {
	v := c.Context().UserValue(customAttributesCtxKey)
	if v == nil {
		c.Context().SetUserValue(customAttributesCtxKey, []slog.Attr{attr})
		return
	}

	switch attrs := v.(type) {
	case []slog.Attr:
		c.Context().SetUserValue(customAttributesCtxKey, append(attrs, attr))
	}
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
