// Package kit is a foundational Go library designed to help developers build consistent,
// and maintainable applications. It provides utilities for structured logging, validation,
// error handling, request parsing, and middleware integration with the Fiber web framework.
//
// # Key Features:
//
//   - Validation: Provides a wrapper around `go-playground/validator`, supporting custom validation tags
//     and localized error messages in Portuguese.
//
//   - Logging: Leverages `slog` for structured, JSON-based logging with support for contextual attributes
//     like request ID, user details, and response status. Includes `LoggerMiddleware` for request-based logging.
//
//   - Middleware: Provides reusable middleware for Fiber, including `LoggerMiddleware` for logging request
//     and response data with contextual information.
//
//   - Request Handling: Simplifies HTTP request parsing and validation through `ParseRequestBody`,
//     reducing boilerplate code for handling and validating request payloads.
//
//   - Testing Utilities: Provides helper functions and types (`Map`, `Request`, `Response`) to facilitate
//     the creation and testing of HTTP handlers.
//
//   - Context Management: Defines `ContextKey` constants to facilitate storing and retrieving metadata
//     (e.g., user info, logging context).
package kit
