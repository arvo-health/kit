package kit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeUnexpectedMessage is the only text an unexpected failure may show to the caller.
const safeUnexpectedMessage = "internal error"

const (
	secretQuery     = "select ssn from patients"
	secretRecordID  = "rec_8f3c1a"
	secretPrivateIP = "10.8.8.8"
	secretDBHost    = "db.internal.svc"
	secretFile      = "/var/lib/kit/secrets/tenant-9f3a.db"
)

// TestUnexpectedErrorsAreNotReturnedToClients is the spec for unexpected-error responses.
// It fails while ErrorHandler copies the raw error string into the JSON body.
func TestUnexpectedErrorsAreNotReturnedToClients(t *testing.T) {
	queryErr := errors.New("pq: " + secretQuery + " where id = '" + secretRecordID + "': relation does not exist")
	dialErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Addr: &net.TCPAddr{
			IP:   net.ParseIP(secretPrivateIP),
			Port: 5432,
		},
		Err: errors.New("connect: connection refused"),
	}
	pathErr := &os.PathError{
		Op:   "open",
		Path: secretFile,
		Err:  os.ErrPermission,
	}

	tests := []struct {
		name           string
		inputError     error
		secrets        []string
		useLoggerMW    bool
		fromMiddleware bool
	}{
		{
			name:       "handler returns a query error",
			inputError: queryErr,
			secrets:    []string{secretQuery, secretRecordID, queryErr.Error()},
		},
		{
			name:       "handler returns a wrapped connection error",
			inputError: fmt.Errorf("connect %s:5432: %w", secretDBHost, dialErr),
			secrets:    []string{secretDBHost, secretPrivateIP, dialErr.Error()},
		},
		{
			name:       "handler returns a joined filesystem and query error",
			inputError: errors.Join(pathErr, queryErr),
			secrets:    []string{secretFile, secretQuery, secretRecordID},
		},
		{
			name:       "handler returns an os.PathError",
			inputError: pathErr,
			secrets:    []string{secretFile, pathErr.Error()},
		},
		{
			name:       "handler returns a net.OpError",
			inputError: dialErr,
			secrets:    []string{secretPrivateIP, dialErr.Error()},
		},
		{
			name:       "handler returns a custom error type",
			inputError: &internalFailure{text: "read " + secretFile + " for record " + secretRecordID},
			secrets:    []string{secretFile, secretRecordID},
		},
		{
			name:           "middleware returns a query error",
			inputError:     queryErr,
			secrets:        []string{secretQuery, secretRecordID, queryErr.Error()},
			fromMiddleware: true,
		},
		{
			name:        "logger middleware still serves the error handler body",
			inputError:  fmt.Errorf("lookup %s: %w", secretDBHost, pathErr),
			secrets:     []string{secretDBHost, secretFile, pathErr.Error()},
			useLoggerMW: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logs := newCaptureLogger()
			status, body := perform(t, logger, tt.inputError, tt.useLoggerMW, tt.fromMiddleware)

			assert.Equal(t, http.StatusInternalServerError, status)
			for _, secret := range tt.secrets {
				assert.NotContains(t, body, secret)
			}

			var parsed map[string]any
			require.NoError(t, json.Unmarshal([]byte(body), &parsed))
			assert.Equal(t, map[string]any{
				"code":        "unknown-error",
				"message":     safeUnexpectedMessage,
				"status_code": float64(http.StatusInternalServerError),
			}, parsed["error"])

			// Logger middleware already records the handler error on its own.
			// The case without it requires ErrorHandler itself to log the original text.
			if !tt.useLoggerMW {
				assert.Contains(t, logs.joined(), tt.inputError.Error())
			}
		})
	}

	t.Run("different unexpected errors share one fixed message", func(t *testing.T) {
		logger, _ := newCaptureLogger()
		_, queryBody := perform(t, logger, queryErr, false, false)
		_, dialBody := perform(t, logger, dialErr, false, false)

		var queryParsed map[string]any
		var dialParsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(queryBody), &queryParsed))
		require.NoError(t, json.Unmarshal([]byte(dialBody), &dialParsed))
		assert.Equal(t, queryParsed["error"], dialParsed["error"])
	})

	t.Run("explicit HTTPError text is returned and wrapper text is not", func(t *testing.T) {
		callerFacing := kit.HTTPNotFoundError("user-not-found", errors.New("user missing"))
		wrapped := fmt.Errorf("open %s for %s: %w", secretFile, secretRecordID, callerFacing)

		logger, _ := newCaptureLogger()
		status, body := perform(t, logger, wrapped, false, false)

		assert.Equal(t, http.StatusNotFound, status)
		assert.NotContains(t, body, secretFile)
		assert.NotContains(t, body, secretRecordID)
		assert.Contains(t, body, "user missing")

		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(body), &parsed))
		assert.Equal(t, map[string]any{
			"code":        "user-not-found",
			"message":     "user missing",
			"status_code": float64(http.StatusNotFound),
		}, parsed["error"])
	})
}

type internalFailure struct {
	text string
}

func (e *internalFailure) Error() string {
	return e.text
}

func perform(t *testing.T, logger *slog.Logger, input error, useLoggerMW bool, fromMiddleware bool) (int, string) {
	t.Helper()

	app := fiber.New(fiber.Config{
		ErrorHandler: kit.ErrorHandler(logger),
	})
	if useLoggerMW {
		app.Use(kit.LoggerMiddleware(logger))
	}
	if fromMiddleware {
		app.Use(func(c *fiber.Ctx) error {
			return input
		})
	}
	app.Get("/records/:id", func(c *fiber.Ctx) error {
		if fromMiddleware {
			return c.SendStatus(http.StatusNoContent)
		}
		return input
	})

	req := httptest.NewRequest(http.MethodGet, "/records/1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw)
}

type captureHandler struct {
	mu     *sync.Mutex
	lines  *[]string
	attrs  []slog.Attr
	groups []string
}

func newCaptureLogger() (*slog.Logger, *captureHandler) {
	handler := &captureHandler{
		mu:    &sync.Mutex{},
		lines: &[]string{},
	}
	return slog.New(handler), handler
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	r = r.Clone()
	var b strings.Builder
	if len(h.groups) > 0 {
		b.WriteString(strings.Join(h.groups, "."))
		b.WriteByte(' ')
	}
	b.WriteString(r.Message)
	for _, attr := range h.attrs {
		b.WriteByte(' ')
		writeAttr(&b, attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		b.WriteByte(' ')
		writeAttr(&b, attr)
		return true
	})

	h.mu.Lock()
	*h.lines = append(*h.lines, b.String())
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.groups = append(append([]string{}, h.groups...), name)
	return &next
}

func (h *captureHandler) joined() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return strings.Join(*h.lines, "\n")
}

func writeAttr(b *strings.Builder, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	b.WriteString(attr.Key)
	b.WriteByte('=')
	if attr.Value.Kind() == slog.KindGroup {
		b.WriteByte('{')
		for _, grouped := range attr.Value.Group() {
			writeAttr(b, grouped)
			b.WriteByte(' ')
		}
		b.WriteByte('}')
		return
	}
	b.WriteString(attr.Value.String())
}
