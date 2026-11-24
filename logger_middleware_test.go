package kit_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// credentialHeaderNames are headers whose values authorize a caller.
// The request denylist already covers some of them. The response denylist
// covers only Set-Cookie. Both logging loops must omit every name here.
var credentialHeaderNames = []string{
	"Authorization",
	"Proxy-Authorization",
	"Authentication",
	"Cookie",
	"Set-Cookie",
	"X-Api-Key",
	"Api-Key",
	"X-Api-Token",
	"X-Access-Token",
	"X-Auth-Token",
	"X-Csrf-Token",
	"X-Xsrf-Token",
}

const (
	requestTraceHeader  = "Accept"
	requestTraceValue   = "visible-request-marker"
	responseTraceHeader = "Cache-Control"
	responseTraceValue  = "visible-response-marker"
)

func placeholder(direction, header string) string {
	return "placeholder-" + direction + "-" + strings.ToLower(header)
}

func TestLoggerMiddlewareDefaultDoesNotLogHeaders(t *testing.T) {
	logText := runLoggedRequest(t, kit.LoggerMiddleware, func(req *http.Request) {
		addCredentialRequestHeaders(req, "default")
	}, func(c *fiber.Ctx) error {
		addCredentialResponseHeaders(c, "default")
		c.Set(responseTraceHeader, responseTraceValue)
		return c.SendStatus(fiber.StatusNoContent)
	})

	require.NotContains(t, logText, requestTraceValue)
	require.NotContains(t, logText, responseTraceValue)
	assertCredentialPlaceholdersAbsent(t, logText, "default")
}

func TestLoggerMiddlewareRequestHeaderLoggingOmitsCredentials(t *testing.T) {
	logText := runLoggedRequest(t, func(logger *slog.Logger) fiber.Handler {
		return kit.LoggerMiddlewareWithConfig(logger, kit.Config{
			DefaultLevel:      slog.LevelInfo,
			WithRequestHeader: true,
		})
	}, func(req *http.Request) {
		req.Header.Set(requestTraceHeader, requestTraceValue)
		addCredentialRequestHeaders(req, "request")
		// A non-canonical name still reaches the middleware after fasthttp
		// normalizes it. Redaction has to follow that normalized name.
		req.Header["x-ApI-kEy"] = append(req.Header["x-ApI-kEy"], placeholder("request", "x-api-key-raw"))
	}, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	require.Contains(t, logText, requestTraceValue, "request header logging did not record an ordinary header")
	assertCredentialPlaceholdersAbsent(t, logText, "request")
	require.NotContains(t, logText, placeholder("request", "x-api-key-raw"))
}

func TestLoggerMiddlewareResponseHeaderLoggingOmitsCredentials(t *testing.T) {
	logText := runLoggedRequest(t, func(logger *slog.Logger) fiber.Handler {
		return kit.LoggerMiddlewareWithConfig(logger, kit.Config{
			DefaultLevel:       slog.LevelInfo,
			WithResponseHeader: true,
		})
	}, func(req *http.Request) {}, func(c *fiber.Ctx) error {
		c.Set(responseTraceHeader, responseTraceValue)
		addCredentialResponseHeaders(c, "response")
		c.Append("x-ApI-kEy", placeholder("response", "x-api-key-extra"))
		return c.SendStatus(fiber.StatusNoContent)
	})

	require.Contains(t, logText, responseTraceValue, "response header logging did not record an ordinary header")
	assertCredentialPlaceholdersAbsent(t, logText, "response")
	require.NotContains(t, logText, placeholder("response", "x-api-key-extra"))
}

func TestLoggerMiddlewareLogsBothHeaderSidesWithoutCredentials(t *testing.T) {
	logText := runLoggedRequest(t, func(logger *slog.Logger) fiber.Handler {
		return kit.LoggerMiddlewareWithConfig(logger, kit.Config{
			DefaultLevel:       slog.LevelInfo,
			WithRequestHeader:  true,
			WithResponseHeader: true,
		})
	}, func(req *http.Request) {
		req.Header.Set(requestTraceHeader, requestTraceValue)
		addCredentialRequestHeaders(req, "both-request")
	}, func(c *fiber.Ctx) error {
		c.Set(responseTraceHeader, responseTraceValue)
		addCredentialResponseHeaders(c, "both-response")
		return c.SendStatus(fiber.StatusNoContent)
	})

	require.Contains(t, logText, requestTraceValue)
	require.Contains(t, logText, responseTraceValue)
	assertCredentialPlaceholdersAbsent(t, logText, "both-request")
	assertCredentialPlaceholdersAbsent(t, logText, "both-response")
}

func addCredentialRequestHeaders(req *http.Request, direction string) {
	for _, name := range credentialHeaderNames {
		req.Header.Add(name, placeholder(direction, name))
		req.Header.Add(name, placeholder(direction, name+"-repeat"))
	}
}

func addCredentialResponseHeaders(c *fiber.Ctx, direction string) {
	for _, name := range credentialHeaderNames {
		c.Set(name, placeholder(direction, name))
		c.Append(name, placeholder(direction, name+"-repeat"))
	}
}

func assertCredentialPlaceholdersAbsent(t *testing.T, logText, direction string) {
	t.Helper()
	for _, name := range credentialHeaderNames {
		require.NotContains(t, logText, placeholder(direction, name), "logged credential header %s", name)
		require.NotContains(t, logText, placeholder(direction, name+"-repeat"), "logged repeated credential header %s", name)
	}
}

func runLoggedRequest(
	t *testing.T,
	middleware func(*slog.Logger) fiber.Handler,
	prepareRequest func(*http.Request),
	handler fiber.Handler,
) string {
	t.Helper()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	app := fiber.New()
	app.Use(middleware(logger))
	app.Get("/probe", handler)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/probe", nil)
	prepareRequest(req)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, resp.Body.Close())
	})
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	return buf.String()
}
