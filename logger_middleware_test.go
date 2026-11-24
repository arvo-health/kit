package kit_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// stdoutMu serializes tests that replace os.Stdout. NewLogger reads that
// file when it builds the JSON handler, which is the sink the middleware uses.
var stdoutMu sync.Mutex

// placeholder values stand in for secrets a caller puts in a link.
// They are not credentials.
const (
	placeholderToken          = "placeholder-token"
	placeholderCode           = "placeholder-code"
	placeholderPassword       = "placeholder-password"
	placeholderEmail          = "placeholder-email"
	placeholderSignature      = "placeholder-signature"
	placeholderTokenUpper     = "placeholder-token-upper"
	placeholderCodeUpper      = "placeholder-code-upper"
	placeholderPasswordUpper  = "placeholder-password-upper"
	placeholderEmailUpper     = "placeholder-email-upper"
	placeholderSignatureUpper = "placeholder-signature-upper"
	placeholderPathToken      = "placeholder-path-token"
	placeholderPathRecord     = "placeholder-path-record"
	placeholderWildcard       = "placeholder-wildcard"
	placeholderPlus           = "placeholder-plus"
	placeholderErrorPath      = "placeholder-error-path"
)

// TestRequestLoggingRedactsSensitiveURLData is the spec for request logging.
// LoggerMiddleware and LoggerMiddlewareWithConfig must keep logging the route
// template, and the log line written by NewLogger must not contain a secret
// carried in the query string or in a route parameter.
func TestRequestLoggingRedactsSensitiveURLData(t *testing.T) {
	middlewares := []struct {
		name string
		new  func(*slog.Logger) fiber.Handler
	}{
		{name: "default", new: kit.LoggerMiddleware},
		{name: "with_config", new: func(logger *slog.Logger) fiber.Handler {
			return kit.LoggerMiddlewareWithConfig(logger, kit.Config{
				DefaultLevel:       slog.LevelInfo,
				WithUserAgent:      true,
				WithRequestHeader:  true,
				WithResponseHeader: true,
			})
		}},
	}

	cases := []struct {
		name     string
		method   string
		target   string
		secrets  []string
		rawQuery string
		route    string
	}{
		{
			name:    "query_token",
			method:  http.MethodGet,
			target:  "/sign-in?" + url.Values{"token": {placeholderToken}}.Encode(),
			secrets: []string{placeholderToken},
			route:   "/sign-in",
		},
		{
			name:    "query_code",
			method:  http.MethodGet,
			target:  "/sign-in?" + url.Values{"code": {placeholderCode}}.Encode(),
			secrets: []string{placeholderCode},
			route:   "/sign-in",
		},
		{
			name:    "query_password",
			method:  http.MethodPost,
			target:  "/sign-in?" + url.Values{"password": {placeholderPassword}}.Encode(),
			secrets: []string{placeholderPassword},
			route:   "/sign-in",
		},
		{
			name:    "query_email",
			method:  http.MethodGet,
			target:  "/sign-in?" + url.Values{"email": {placeholderEmail}}.Encode(),
			secrets: []string{placeholderEmail},
			route:   "/sign-in",
		},
		{
			name:    "query_signature",
			method:  http.MethodGet,
			target:  "/sign-in?" + url.Values{"signature": {placeholderSignature}}.Encode(),
			secrets: []string{placeholderSignature},
			route:   "/sign-in",
		},
		{
			name:   "query_names_case_insensitive",
			method: http.MethodGet,
			target: "/sign-in?" + url.Values{
				"Token":     {placeholderTokenUpper},
				"CODE":      {placeholderCodeUpper},
				"Password":  {placeholderPasswordUpper},
				"Email":     {placeholderEmailUpper},
				"SIGNATURE": {placeholderSignatureUpper},
			}.Encode(),
			secrets: []string{
				placeholderTokenUpper,
				placeholderCodeUpper,
				placeholderPasswordUpper,
				placeholderEmailUpper,
				placeholderSignatureUpper,
			},
			route: "/sign-in",
		},
		{
			name:   "query_combined",
			method: http.MethodGet,
			target: "/sign-in?" + combinedSecretQuery(),
			secrets: []string{
				placeholderToken,
				placeholderCode,
				placeholderPassword,
				placeholderEmail,
				placeholderSignature,
			},
			rawQuery: combinedSecretQuery(),
			route:    "/sign-in",
		},
		{
			name:    "path_token",
			method:  http.MethodGet,
			target:  "/reset/" + placeholderPathToken,
			secrets: []string{placeholderPathToken},
			route:   "/reset/:token",
		},
		{
			name:    "path_record_id",
			method:  http.MethodGet,
			target:  "/records/" + placeholderPathRecord + "?" + url.Values{"token": {placeholderToken}}.Encode(),
			secrets: []string{placeholderPathRecord, placeholderToken},
			route:   "/records/:id",
		},
		{
			name:    "wildcard_param",
			method:  http.MethodGet,
			target:  "/files/" + placeholderWildcard,
			secrets: []string{placeholderWildcard},
			route:   "/files/*",
		},
		{
			name:    "plus_param",
			method:  http.MethodGet,
			target:  "/blob/" + placeholderPlus,
			secrets: []string{placeholderPlus},
			route:   "/blob/+",
		},
	}

	for _, mw := range middlewares {
		t.Run(mw.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					logs := captureStdout(t, func() {
						logger := kit.NewLogger(slog.LevelInfo)
						app := fiber.New()
						app.Use(mw.new(logger))
						registerLoggingRoutes(app)

						req := httptest.NewRequest(tc.method, tc.target, nil)
						req.Header.Set("User-Agent", "kit-test")
						resp, err := app.Test(req, -1)
						require.NoError(t, err)
						defer resp.Body.Close() //nolint:errcheck // Closing the test response body is best-effort.
						require.Equal(t, http.StatusNoContent, resp.StatusCode)
					})

					require.NotEmpty(t, logs)
					require.Contains(t, logs, `"method":"`+tc.method+`"`)
					require.Contains(t, logs, tc.route)
					if tc.rawQuery != "" {
						require.NotContains(t, logs, tc.rawQuery)
					}
					for _, secret := range tc.secrets {
						require.NotContains(t, logs, secret)
					}
				})
			}
		})
	}
}

// TestErrorHandlerDoesNotRecordSensitivePath covers the fiber error logger,
// which records the request path on its own.
func TestErrorHandlerDoesNotRecordSensitivePath(t *testing.T) {
	logs := captureStdout(t, func() {
		logger := kit.NewLogger(slog.LevelInfo)
		app := fiber.New(fiber.Config{
			ErrorHandler: kit.ErrorHandler(logger),
		})
		app.Get("/sessions/:code", func(*fiber.Ctx) error {
			return fiber.NewError(fiber.StatusBadRequest, "rejected")
		})

		target := "/sessions/" + placeholderErrorPath + "?" + url.Values{"token": {placeholderToken}}.Encode()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close() //nolint:errcheck // Closing the test response body is best-effort.
		require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	require.Contains(t, logs, `"method":"GET"`)
	require.Contains(t, logs, "request failed: rejected")
	require.NotContains(t, logs, placeholderErrorPath)
	require.NotContains(t, logs, placeholderToken)
}

func registerLoggingRoutes(app *fiber.App) {
	noContent := func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	}
	app.Get("/sign-in", noContent).Name("Sign in")
	app.Post("/sign-in", noContent).Name("Sign in")
	app.Get("/reset/:token", noContent).Name("Reset")
	app.Get("/records/:id", noContent).Name("Get record")
	app.Get("/files/*", noContent).Name("Get file")
	app.Get("/blob/+", noContent).Name("Get blob")
}

func combinedSecretQuery() string {
	return url.Values{
		"token":     {placeholderToken},
		"code":      {placeholderCode},
		"password":  {placeholderPassword},
		"email":     {placeholderEmail},
		"signature": {placeholderSignature},
		"page":      {"listing"},
	}.Encode()
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	fn()

	os.Stdout = original
	require.NoError(t, writer.Close())

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	return buf.String()
}
