package kit

import (
	"context"
	"log/slog"
	"sync"
)

// MockLogHandler is a mocked log handler for capturing and testing log records.
type MockLogHandler struct {
	mu      sync.Mutex
	records []slog.Record
	level   slog.Level
	attrs   []slog.Attr
	groups  []string
}

// NewMockLogHandlerWithLevel creates a new MockLogHandler with a specified minimum log level.
func NewMockLogHandlerWithLevel(level slog.Level) *MockLogHandler {
	return &MockLogHandler{
		level:   level,
		records: []slog.Record{},
		attrs:   []slog.Attr{},
		groups:  []string{},
	}
}

// NewMockLogHandler creates a new MockLogHandler with the default log level set to LevelInfo.
func NewMockLogHandler() *MockLogHandler {
	return NewMockLogHandlerWithLevel(slog.LevelInfo)
}

// Handle processes a log record by appending it to the handler's records and applying contextual attributes.
func (h *MockLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	r.AddAttrs(h.attrs...)
	h.records = append(h.records, r)
	return nil
}

// Enabled determines if logging is enabled for the given context and log level based on the handler's minimum level.
func (h *MockLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// CapturedRecords returns a copy of the captured log records stored in the handler.
func (h *MockLogHandler) CapturedRecords() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.records
}

// WithAttrs returns a new handler instance with the provided attributes appended to the existing attributes.
func (h *MockLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	newHandler := *h //nolint:govet // Test purpose
	newHandler.attrs = append(h.attrs, attrs...)
	return &newHandler
}

// WithGroup creates and returns a new handler with the provided group name appended to the existing group hierarchy.
func (h *MockLogHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	newHandler := *h //nolint:govet // Test purpose
	newHandler.groups = append(h.groups, name)
	return &newHandler
}
