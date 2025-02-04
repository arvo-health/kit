# Kit

**Kit** is a foundational Go library designed to help developers build consistent, and maintainable applications.  
It provides utilities for structured logging, validation, error handling, and middleware integration for the **Fiber** web framework.

## Features

- **Validation**: Simplifies struct validation with a centralized system that generates detailed, localized (pt_BR) error messages.  
  It is a wrapper around `go-playground/validator` with support for custom validation tags.
- **Error Handling**: Standardizes application error responses with `ResponseError`, enabling structured, reusable error handling.
- **Fiber Integration**: Provides middleware like `LoggerMiddleware` for structured request logging and `ErrorHandler` for error processing.
- **Structured Logging**: Uses `slog` for JSON-based, context-rich logging with support for metadata such as request IDs, user details, and response status.
- **Middleware**: Includes reusable middleware, such as request logging for Fiber applications.



## Package Overview

```plaintext
kit/
├── context_key.go         # Defines context keys for storing metadata
├── error_handler.go       # Custom Fiber error handler for structured error responses
├── error_response.go      # Standardized error response structure
├── logger.go              # Structured logging utilities using slog
├── logger_middleware.go   # Middleware for request logging in Fiber
├── validator.go           # Wrapper for struct validation with localized messages
├── validator_error.go     # Custom validation error structure
├── doc.go                 # General package documentation
```

## Installation

To install the package, run:

```sh
go get github.com/arvo-health/kit
```

Then, import it in your Go application:
    
```go
import "github.com/arvo-health/kit"
```

## Usage Examples

### 1. Logging with Middleware

Use the `LoggerMiddleware` to log structured request and response data.

```go
package main

import (
    "github.com/arvo-health/kit"
    "github.com/gofiber/fiber/v2"
    "log/slog"
)

func main() {
    logger := kit.NewLogger(slog.LevelInfo,
        slog.Group("service",
            slog.String("name", "my-service"),
            slog.String("env", "prod"),
            slog.String("version", "v1.0.0"),
        ),
    )

    app := fiber.New()
    app.Use(kit.LoggerMiddleware(logger))

    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendString("Hello, world!")
    })

    app.Listen(":8080")
}

```

### 2. Validation with Error Handling

Use the kit.NewValidator() for struct validation, along with ErrorHandler() for centralized error handling.

```go
package main

import (
    "errors"
    "github.com/arvo-health/kit"
    "github.com/gofiber/fiber/v2"
    "net/http"
)

func main() {
    app := fiber.New(fiber.Config{
        ErrorHandler: kit.ErrorHandler(),
    })

    app.Post("/validate", func(c *fiber.Ctx) error {
        type User struct {
            Name   string `validate:"required" custom:"Nome"`
            Email  string `validate:"required,email" custom:"E-mail"`
            Age    int    `validate:"gte=18" custom:"Idade"`
            Gender string `validate:"required,oneof=M F" custom:"Gênero"`
        }

        validator, _ := kit.NewValidator()
        
        user := User{Email: "invalid-email"}
        if err := c.BodyParser(&user); err != nil {
            return kit.NewResponseError(err, "INVALID_REQUEST", http.StatusBadRequest)
        }

        if err := validator.Validate(user); err != nil {
            return kit.NewResponseError(err, "USER_VALIDATION", http.StatusUnprocessableEntity)
        }

        return c.SendStatus(http.StatusOK)
    })

    app.Listen(":8080")
}
```

**Expected Response for Invalid Data:**

```json
// POST /validate
// 422 Unprocessable Entity
{
  "error": {
    "code": "USER_VALIDATION",
    "message": "validation failed: user validation failed",
    "details": {
      "Email": "Email deve ser um endereço de e-mail válido",
      "Gênero": "Gênero deve ser um de [M F]",
      "Idade": "Idade deve ser 18 ou superior",
      "Nome": "Nome é um campo obrigatório"
    }
  }
}
```

The example below demonstrates how to perform **complex** validation.

```go
package main

import (
    "errors"
    "github.com/arvo-health/kit"
    "github.com/gofiber/fiber/v2"
    "net/http"
)

func main() {
    app := fiber.New(fiber.Config{
        ErrorHandler: kit.ErrorHandler(),
    })

    app.Post("/validate", func(c *fiber.Ctx) error {
        type Analysis struct {
            Status      string
            Description string
        }

        analysis := Analysis{Status: "DENIED"}

        // Create a new validator.Error instance.
        validatorErr := kit.NewValidatorError("analysis validation failed")

        // perform the validation
        if analysis.Status == "DENIED" && len(analysis.Description) < 5 {
            validatorErr.AddValidation("Descrição", "Descrição deve ter pelo menos 5 caracteres")
        }

        // perform more complex validation
        if true {
            // Add more field-err validation error.
            validatorErr.AddValidation("Status", "Status deve ser OK")
        }

        // Check if the validatorErr has any validations.
        if validatorErr.HasValidations() {
            // Return a new *ResponseError wrapping it
            return kit.NewResponseError(validatorErr, "ANALYSIS_VALIDATION", http.StatusUnprocessableEntity)
        }

        return c.SendStatus(http.StatusOK)
    })

    app.Listen(":8080")
}
```
**Expected Response for validation:**

```json
// POST /validate
// 400 Unprocessable Entity
{
  "error": {
    "code": "ANALYSIS_VALIDATION",
    "message": "analysis validation failed",
    "details": {
      "Descrição": "Descrição deve ter pelo menos 5 caracteres",
      "Status": "Status deve ser xpto"
    }
  }
}
```

The field `details` only exist when the error is a **validation** error.

### 3. Custom Error Handling

Use `kit.NewResponseError()` to create structured errors.

```go
package main

import (
    "errors"
    "github.com/arvo-health/kit"
    "github.com/gofiber/fiber/v2"
    "net/http"
)

var ErrUserNotFound = errors.New("user not found")

func main() {
    app := fiber.New(fiber.Config{
        ErrorHandler: kit.ErrorHandler(),
    })

    app.Get("/user/:id", func(c *fiber.Ctx) error {
        return kit.NewResponseError(ErrUserNotFound, "USER_NOT_FOUND", http.StatusNotFound)
    })

    app.Listen(":8080")
}
```

**Expected Response for Not Found Error:**

```json
// GET /user/123
// 404 Not Found
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "user not found"
  }
}
```

## Contributions

Contributions and feedback are welcome!
