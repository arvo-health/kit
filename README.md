# Kit: Go Utilities for Building Consistent and Maintainable Applications

Kit is a foundational Go library designed to help developers build consistent and maintainable
applications. It provides utilities for structured logging, validation, error handling, request parsing, 
and middleware integration with the Fiber web framework.

## Key Features:

  - **Validation**: Provides a wrapper around `go-playground/validator`, supporting custom validation tags
    and localized error messages in Portuguese.

  - **Error Handling**: Standardizes application errors with structured responses through `ResponseError`.
    Includes a Fiber-compatible error handler (`ErrorHandler`) for seamless error processing.
    Provides structured error types (`Error` and `ValidationErrors`) for better debugging.

  - **Logging**: Leverages `slog` for structured, JSON-based logging with support for contextual attributes
    like request ID, user details, and response status. Includes `LoggerMiddleware` for request-based logging.

  - **Middleware**: Provides reusable middleware for Fiber, including `LoggerMiddleware` for logging request
    and response data with contextual information.

  - **Request Handling**: Simplifies HTTP request parsing and validation through `ParseRequestBody`,
    reducing boilerplate code for handling and validating request payloads.

  - **Testing Utilities**: Provides helper functions and types (`Map`, `Request`, `Response`) to facilitate
    the creation and testing of HTTP handlers.

    - **Mock de Logging**: Proporciona uma implementação de log (slog) mockada (`MockLogHandler`) para capturar e testar mensagens de log em cenários de teste.


  - **Context Management**: Defines `ContextKey` constants to facilitate storing and retrieving metadata
    (e.g., user info, logging context).

## Package Structure:

```plaintext
kit/
├── context_key.go            # Context keys for storing metadata
├── error_handler.go          # Error handler for Fiber
├── http_error.go             # HTTPError structure and utility functions
├── handler_utils.go          # Utilities for managing HTTP requests
├── logger.go                 # Structured logging utilities
├── logger_middleware.go      # Middleware for Fiber request logging
├── validator.go              # Validation wrapper with localized messages
├── validator_error.go        # Custom validation error structure
├── healthcheck_middleware.go # Middleware for health check endpoints
├── test_utils.go             # HTTP handler testing utilities
├── test_slog_mock.go         # Mock de handler de log para testes
```

## Basic Usage Examples

### **1. Error Handling and Structured Logging**
Use `kit.LoggerMiddleware()` to log detailed, structured responses, and use `HTTPError`, `ErrorHandler`, and other utilities for consistent error management.

```go
package main

import (
	"fmt"
	"log/slog"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Initialize logger with service-level context
	logger := kit.NewLogger(slog.LevelInfo,
		slog.Group("service",
			slog.String("name", "my-service"),
			slog.String("env", "prod"),
			slog.String("version", "v1.0.0"),
		),
	)

	app := fiber.New(fiber.Config{
		// Configure kit.ErrorHandler as the central error handler
		ErrorHandler: kit.ErrorHandler(),
	})

	// Add logger middleware
	app.Use(kit.LoggerMiddleware(logger))

	// Simple endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, world!")
	}).Name("Hello world")

	// Endpoint with error
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		// Returning an HTTP structured error
		return kit.HTTPNotFoundError("user-not-found",
			fmt.Errorf("user with ID %s not found", id))
	}).Name("Get User")

	app.Listen(":8080")
}
```

### **2. Validating Payloads**

`kit.ParseRequestBody` simplifies the processing of JSON payloads in Fiber, automatically validating them and returning standardized error responses on failure.
See the official [go-playground/validator](https://pkg.go.dev/github.com/go-playground/validator/v10#section-readme) documentation for more tag validation options.

```go
package main

import (
	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
)

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required" custom:"Nome"`
	Email string `json:"email" validate:"required,email" custom:"Email"`
}

func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: kit.ErrorHandler(),
	})

	validator := kit.NewValidator()

	app.Post("/users", func(c *fiber.Ctx) error {
		var req CreateUserRequest

		// Parse and validate the request body
		if err := kit.ParseRequestBody(&req, c, validator); err != nil {
			return err // Structured error will be handled by ErrorHandler
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"name":  req.Name,
			"email": req.Email,
		})
	})

	app.Listen(":8080")
}
```

### **3. Logging Mock**

The `MockLogHandler` is for capturing and testing logged messages in the context of testing. It allows validating if certain messages were logged.

```go
package main

import (
	"log/slog"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

func TestMockLogHandler(t *testing.T) {
	// Initializes the MockLogHandler to capture logs
	handler := kit.NewMockLogHandler()

	// Creates a logger with the mock handler
	logger := slog.New(handler)

	// Logs an example message
	logger.Info("Example informational log", slog.String("key", "value"))

	// Retrieves the captured records
	records := handler.CapturedRecords()

	// Asserts that there is exactly one log record captured
	assert.Len(t, records, 1, "Expected 1 log record, but found %d", len(records))

	// Asserts that the captured log level matches the expected INFO level
	assert.Equal(t, slog.LevelInfo, records[0].Level, "Expected log level to be INFO, but found %s", records[0].Level)

	// Assert that the log message content matches the expected value
	assert.Equal(t, "Example informational log", records[0].Message, "Incorrect log message: expected 'Example informational log', got '%s'", records[0].Message)
}
```

### **4. Health Check Middleware**
Add liveness and readiness health check endpoints easily to your applications.

```go
package main

import (
	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	// Use middleware for health checks
	app.Group("/healthcheck", kit.HealthCheckMiddleware())

	app.Listen(":8080")
}
```

- Endpoints:
    - **Liveness**: `/healthcheck/live`
    - **Readiness**: `/healthcheck/ready`

## **Example of Structured Logs**
#### Log Generated by `LoggerMiddleware`:

```json
{
  "time": "2025-04-09T14:58:39.225459-03:00",
  "level": "WARN",
  "msg": "Get User: request failed: User with ID 10 not found",
  "service": {
    "name": "my-service",
    "env": "prod",
    "version": "v1.0.0"
  },
  "request_id": "dae8c97b-f8bb-4b1a-a5a9-2608912ad605",
  "request": {
    "time": "2025-04-09T17:58:39.225248Z",
    "method": "GET",
    "host": "localhost:8080",
    "path": "/users/10",
    "query": "",
    "params": {
      "id": "10"
    },
    "route": "/users/:id",
    "length": 0
  },
  "response": {
    "time": "2025-04-09T17:58:39.225455Z",
    "latency": 207000,
    "status": 404,
    "length": 91
  },
  "user": {
    "email": "unknown",
    "company": "unknown",
    "company_category": "unknown",
    "permissions": "<nil>"
  }
}
```

## **Example of response error**

```json
{
  "error": {
    "code": "request-validation",
    "message": "validation failed",
    "details": [
      "Email é um campo obrigatório"
    ],
    "status_code": 400
  }
}
```
The field details only exist when the error is a **validation** error.


## Installation

To install the package, run:

```shell
  go get github.com/arvo-health/kit
```

Then, import it in your Go application:

```go
import "github.com/arvo-health/kit"
```

# Contribution

Contributions and feedback are welcome!
