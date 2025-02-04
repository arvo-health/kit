// Package kit is a foundational Go library designed to help developers build robust,
// consistent, and maintainable applications. It provides utilities for structured logging,
// validation, error handling, and middleware integration with the Fiber web framework.
//
// # Key Features:
//
//   - Validation: Provides a wrapper around `go-playground/validator`, supporting custom validation tags
//     and localized error messages in Portuguese.
//
//   - Error Handling: Standardizes application errors with structured responses through `ResponseError`.
//     Includes a Fiber-compatible error handler for seamless error processing.
//
//   - Logging: Leverages `slog` for structured, JSON-based logging with support for contextual attributes
//     like request ID, user details, and response status.
//
//   - Middleware: Provides reusable middleware for Fiber, such as `LoggerMiddleware` for logging request
//     and response data.
//
//   - Context Management: Defines `ContextKey` constants to facilitate storing and retrieving metadata
//     (e.g., user info, logging context).
package kit
