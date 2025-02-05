// Package kit provides structured logging utilities for Go applications.
// This file defines a logger using `slog` for JSON-based structured logs.

package kit

import (
	"log/slog"
	"os"
)

// NewLogger creates a new instance of a JSON-based `slog.Logger` with customizable attributes.
// It allows adding additional context (e.g., service name, version) to logs.
//
// Parameters:
// - `level`: Specifies the logging level (e.g., Info, Debug, Error).
// - `opts`: Additional attributes for contextual logging.
//
// Returns:
// - A pointer to the configured `slog.Logger` instance.
func NewLogger(level slog.Level, opts ...slog.Attr) *slog.Logger {
	// Configure a handler for JSON-formatted logs with source code information.
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	// Create the logger with the configured handler.
	logger := slog.New(handler)

	// Attach additional context (if provided) to the logger.
	for _, opt := range opts {
		logger = logger.With(opt)
	}

	// Set this logger as the default logger for the application.
	slog.SetDefault(logger)
	return logger
}
