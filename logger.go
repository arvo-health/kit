// Package kit provides structured logging utilities for Go applications.
// This file defines a logger utility using `slog` for JSON-based structured logs,
// allowing customization and attaching additional context for better debugging and monitoring.

package kit

import (
	"log/slog"
	"os"
)

// NewLogger creates a new instance of a JSON-based `slog.Logger` with customizable attributes.
// It allows adding additional context (e.g., service name, version) to logs.
func NewLogger(level slog.Level, opts ...slog.Attr) *slog.Logger {
	// Configure a handler for JSON-formatted logs with source code information.
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	// Create the logger with the configured handler and attach additional context (if provided) to the logger.
	logger := slog.New(handler)
	for _, opt := range opts {
		logger = logger.With(opt)
	}

	return logger
}
